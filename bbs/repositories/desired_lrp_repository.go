package repositories

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/go-gorp/gorp"
)

type desired struct {
	Id      int64 `db:"id"`
	Version int64 `db:"version"`

	ProcessGuid string `db:"process_guid"`
	Domain      string `db:"domain"`
	Payload     string `db:"payload"`
}

//go:generate counterfeiter -o fake_repositories/fake_desired_lrp_repository.go . DesiredLRPRepository
type DesiredLRPRepository interface {
	Create(sql gorp.SqlExecutor, modelDesired models.DesiredLRP) (models.DesiredLRP, error)
	GetAll(sql gorp.SqlExecutor) ([]models.DesiredLRP, error)
	GetByProcessGuid(sql gorp.SqlExecutor, guid string) (models.DesiredLRP, int64, error)
	GetAllByDomain(sql gorp.SqlExecutor, domain string) ([]models.DesiredLRP, error)
	DeleteByProcessGuid(sql gorp.SqlExecutor, guid string) error
	UpdateDesiredLRP(sql gorp.SqlExecutor, guid string, updateRequest models.DesiredLRPUpdate) (models.DesiredLRP, error)
}

type desiredRepository struct {
}

func NewDesiredLRPRepository(dbmap *gorp.DbMap) (DesiredLRPRepository, error) {
	dbmap.AddTableWithName(desired{}, "desired_lrps").SetKeys(true, "Id").SetVersionCol("Version")

	_, err := dbmap.Exec("CREATE TABLE IF NOT EXISTS desired_lrps (" +
		"id bigint(20) NOT NULL AUTO_INCREMENT, PRIMARY KEY (id)," +
		"`version` bigint(20) DEFAULT NULL," +
		"process_guid varchar(255) NOT NULL UNIQUE," +
		"domain varchar(255) NOT NULL," +
		"payload longtext DEFAULT NULL" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8")

	return &desiredRepository{}, err
}

func (repo *desiredRepository) Create(sql gorp.SqlExecutor, model models.DesiredLRP) (models.DesiredLRP, error) {
	desired, err := desiredFromModel(model)
	if err != nil {
		return models.DesiredLRP{}, err
	}

	err = sql.Insert(&desired)
	if err != nil {
		return models.DesiredLRP{}, err
	}

	return modelFromDesired(&desired)
}

func (repo *desiredRepository) DeleteByProcessGuid(sql gorp.SqlExecutor, guid string) error {
	id, index, err := primaryKeyAndIndexForDesiredProcessGuid(sql, guid)
	if err != nil {
		return shared.ConvertStoreError(err)
	}

	desired := desired{Id: id, Version: index}
	_, err = sql.Delete(&desired)

	return err
}

func (repo *desiredRepository) UpdateDesiredLRP(sql gorp.SqlExecutor, guid string, updateRequest models.DesiredLRPUpdate) (models.DesiredLRP, error) {
	d := desired{}
	err := sql.SelectOne(&d, "select * from desired_lrps where process_guid = ?", guid)
	if err != nil {
		return models.DesiredLRP{}, shared.ConvertStoreError(err)
	}

	model, err := modelFromDesired(&d)
	if err != nil {
		return models.DesiredLRP{}, shared.ConvertStoreError(err)
	}

	if updateRequest.Annotation != nil {
		model.Annotation = *updateRequest.Annotation
	}

	model.Routes = updateRequest.Routes

	if updateRequest.Instances != nil {
		model.Instances = *updateRequest.Instances
	}

	primaryKey := d.Id

	d, err = desiredFromModel(model)
	if err != nil {
		return models.DesiredLRP{}, shared.ConvertStoreError(err)
	}
	d.Id = primaryKey

	_, err = sql.Update(&d)
	if err != nil {
		return models.DesiredLRP{}, shared.ConvertStoreError(err)
	}

	return modelFromDesired(&d)
}

func (repo *desiredRepository) GetAll(sql gorp.SqlExecutor) ([]models.DesiredLRP, error) {
	lrps, err := sql.Select(desired{}, "select * from desired_lrps")
	if err != nil {
		return nil, err
	}

	return desiredLRPsToModels(lrps)
}

func (repo *desiredRepository) GetByProcessGuid(sql gorp.SqlExecutor, guid string) (models.DesiredLRP, int64, error) {
	d := desired{}
	err := sql.SelectOne(&d, "select * from desired_lrps where process_guid = ?", guid)
	if err != nil {
		return models.DesiredLRP{}, 0, shared.ConvertStoreError(err)
	}

	model, err := modelFromDesired(&d)
	if err != nil {
		return models.DesiredLRP{}, 0, shared.ConvertStoreError(err)
	}

	return model, d.Version, nil
}

func (repo *desiredRepository) GetAllByDomain(sql gorp.SqlExecutor, domain string) ([]models.DesiredLRP, error) {
	lrps, err := sql.Select(desired{}, "select * from desired_lrps where domain = ?", domain)
	if err != nil {
		return nil, err
	}

	return desiredLRPsToModels(lrps)
}

func desiredFromModel(model models.DesiredLRP) (desired, error) {
	payload, err := model.MarshalJSON()
	if err != nil {
		return desired{}, err
	}

	return desired{
		ProcessGuid: model.ProcessGuid,
		Domain:      model.Domain,
		Payload:     string(payload),
	}, nil
}

func modelFromDesired(desired *desired) (models.DesiredLRP, error) {
	model := models.DesiredLRP{}

	err := model.UnmarshalJSON([]byte(desired.Payload))
	if err != nil {
		return models.DesiredLRP{}, err
	}

	return model, nil
}

func desiredLRPsToModels(lrps []interface{}) ([]models.DesiredLRP, error) {
	models := make([]models.DesiredLRP, 0, len(lrps))
	for _, l := range lrps {
		if desired, ok := l.(*desired); ok {
			model, err := modelFromDesired(desired)
			if err != nil {
				return nil, err
			}
			models = append(models, model)
		}
	}

	return models, nil
}

func primaryKeyAndIndexForDesiredProcessGuid(sql gorp.SqlExecutor, guid string) (int64, int64, error) {
	desired := desired{}

	err := sql.SelectOne(&desired, "select desired_lrps.id, desired_lrps.version from desired_lrps where process_guid = ?", guid)
	if err != nil {
		return 0, 0, err
	}

	return desired.Id, desired.Version, nil
}
