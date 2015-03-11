package repositories

import (
	"github.com/cloudfoundry-incubator/runtime-schema/bbs/shared"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/go-gorp/gorp"
)

type actual struct {
	Id      int64 `db:"id"`
	Version int64 `db:"version"`

	ProcessGuid string `db:"process_guid"`
	Payload     string `db:"payload"`
}

//go:generate counterfeiter -o fake_repositories/fake_actual_lrp_repository.go . ActualLRPRepository
type ActualLRPRepository interface {
	Create(sql gorp.SqlExecutor, modelActual models.ActualLRP) (models.ActualLRP, error)
	GetByProcessGuid(sql gorp.SqlExecutor, guid string) (models.ActualLRP, int64, error)
}

type actualRepository struct {
}

func NewActualLRPRepository(dbmap *gorp.DbMap) (ActualLRPRepository, error) {
	dbmap.AddTableWithName(actual{}, "actual_lrps").SetKeys(true, "Id").SetVersionCol("Version")

	_, err := dbmap.Exec("CREATE TABLE IF NOT EXISTS actual_lrps (" +
		"id bigint(20) NOT NULL AUTO_INCREMENT, PRIMARY KEY (id)," +
		"version bigint(20) DEFAULT NULL," +
		"process_guid varchar(255) NOT NULL UNIQUE," +
		"payload longtext DEFAULT NULL" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8")

	return &actualRepository{}, err
}

func (repo *actualRepository) Create(sql gorp.SqlExecutor, model models.ActualLRP) (models.ActualLRP, error) {
	actual, err := actualFromModel(model)
	if err != nil {
		return models.ActualLRP{}, err
	}

	err = sql.Insert(&actual)
	if err != nil {
		return models.ActualLRP{}, err
	}

	return modelFromActual(&actual)
}

func (repo *actualRepository) GetByProcessGuid(sql gorp.SqlExecutor, guid string) (models.ActualLRP, int64, error) {
	a := actual{}
	err := sql.SelectOne(&a, "select * from actual_lrps where process_guid = ?", guid)
	if err != nil {
		return models.ActualLRP{}, 0, shared.ConvertStoreError(err)
	}

	model, err := modelFromActual(&a)
	if err != nil {
		return models.ActualLRP{}, 0, shared.ConvertStoreError(err)
	}

	return model, a.Version, nil
}

func actualFromModel(model models.ActualLRP) (actual, error) {
	payload, err := models.ToJSON(model)
	if err != nil {
		return actual{}, err
	}

	return actual{
		ProcessGuid: model.ProcessGuid,
		Payload:     string(payload),
	}, nil
}

func modelFromActual(actual *actual) (models.ActualLRP, error) {
	model := models.ActualLRP{}

	err := models.FromJSON([]byte(actual.Payload), &model)
	if err != nil {
		return models.ActualLRP{}, err
	}

	return model, nil
}
