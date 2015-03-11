package models

type VolumeSet struct {
	VolumeSetGuid    string `json:"volume_set_guid"`
	Stack            string `json:"stack"`
	Instances        int    `json:"instances"`
	SizeMB           int    `json:"size_mb"`
	ReservedMemoryMB int    `json:"reserved_memory_mb"`
	Since            int64  `json:"since"`
}

func (v VolumeSet) Validate() error {
	var validationError ValidationError

	if v.VolumeSetGuid == "" {
		validationError = validationError.Append(ErrInvalidField{"volume_set_guid"})
	}
	if v.Stack == "" {
		validationError = validationError.Append(ErrInvalidField{"stack"})
	}
	if v.Instances < 1 {
		validationError = validationError.Append(ErrInvalidField{"instances"})
	}
	if v.SizeMB < 1 {
		validationError = validationError.Append(ErrInvalidField{"size_mb"})
	}
	if v.ReservedMemoryMB < 1 {
		validationError = validationError.Append(ErrInvalidField{"reserved_memory_mb"})
	}

	if !validationError.Empty() {
		return validationError
	}

	return nil
}

type VolumeSetAttachment struct {
	VolumeSetGuid string `json:"volume_set_guid"`
	Path          string `json:"path"`
}

func (v VolumeSetAttachment) Validate() error {
	var validationError ValidationError

	if v.VolumeSetGuid == "" {
		validationError = validationError.Append(ErrInvalidField{"volume_set_guid"})
	}
	if v.Path == "" {
		validationError = validationError.Append(ErrInvalidField{"path"})
	}

	if !validationError.Empty() {
		return validationError
	}

	return nil
}

type VolumeState string

const (
	VolumeStatePending VolumeState = "Pending"
	VolumeStateRunning VolumeState = "Running"
	VolumeStateFailed  VolumeState = "Failed"
)

type Volume struct {
	VolumeSetGuid    string      `json:"volume_set_guid"`
	VolumeGuid       string      `json:"volume_guid"`
	CellID           string      `json:"cell_id"`
	Index            int         `json:"index"`
	SizeMB           int         `json:"size_mb"`
	ReservedMemoryMB int         `json:"reserved_memory_mb"`
	State            VolumeState `json:"state"`
	PlacementError   string      `json:"placement_error,omitempty"`
	Since            int64       `json:"since"`
}

func (v Volume) Validate() error {
	var validationError ValidationError
	if v.VolumeSetGuid == "" {
		validationError = validationError.Append(ErrInvalidField{"volume_set_guid"})
	}
	if v.SizeMB < 1 {
		validationError = validationError.Append(ErrInvalidField{"size_mb"})
	}
	if v.ReservedMemoryMB < 1 {
		validationError = validationError.Append(ErrInvalidField{"reserved_memory_mb"})
	}

	if !validationError.Empty() {
		return validationError
	}

	return nil
}

type VolumeStartRequest struct {
	VolumeSet VolumeSet `json:"volume_set"`
	Indices   []uint    `json:"indices"`
}

func (v VolumeStartRequest) Validate() error {
	var validationError ValidationError

	err := v.VolumeSet.Validate()
	if err != nil {
		validationError = validationError.Append(err)
	}

	if len(v.Indices) == 0 {
		validationError = validationError.Append(ErrInvalidField{"indices"})
	}

	if !validationError.Empty() {
		return validationError
	}

	return nil
}
