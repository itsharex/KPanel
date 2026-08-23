package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"regexp"
	"time"
	"unicode"
	"unicode/utf8"
)

const DiskMountPointMaxBytes = 4096

const (
	DiskPartitionActionMount   = "mount"
	DiskPartitionActionUnmount = "unmount"
	DiskPartitionActionFormat  = "format"
	DiskPartitionActionCheck   = "check"
	DiskPartitionActionRepair  = "repair"
)

const (
	DiskFilesystemExt4 = "ext4"
	DiskFilesystemXFS  = "xfs"
	DiskFilesystemNTFS = "ntfs"
	DiskFilesystemVFAT = "vfat"
)

const (
	DiskPartitionJobQueued         = "queued"
	DiskPartitionJobRunning        = "running"
	DiskPartitionJobSucceeded      = "succeeded"
	DiskPartitionJobFailed         = "failed"
	DiskPartitionJobNeedsAttention = "needs_attention"
)

var diskOpaqueIDPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

type DiskPlatform struct {
	Kind     string `json:"kind"`
	Label    string `json:"label"`
	Writable bool   `json:"writable"`
	Reason   string `json:"reason,omitempty"`
}

type DiskFilesystem struct {
	Type     string `json:"type"`
	Version  string `json:"version,omitempty"`
	Label    string `json:"label,omitempty"`
	UUID     string `json:"uuid,omitempty"`
	PartUUID string `json:"partUuid,omitempty"`
}

type DiskMount struct {
	Path           string   `json:"path"`
	Persistent     bool     `json:"persistent"`
	TotalBytes     *uint64  `json:"totalBytes,omitempty"`
	UsedBytes      *uint64  `json:"usedBytes,omitempty"`
	AvailableBytes *uint64  `json:"availableBytes,omitempty"`
	UsagePercent   *float64 `json:"usagePercent,omitempty"`
}

type DiskOperationState struct {
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason,omitempty"`
}

type DiskDevice struct {
	ID                string                        `json:"id"`
	Path              string                        `json:"path"`
	Name              string                        `json:"name"`
	Type              string                        `json:"type"`
	ParentID          string                        `json:"parentId,omitempty"`
	SizeBytes         uint64                        `json:"sizeBytes"`
	ReadOnly          bool                          `json:"readOnly"`
	Removable         bool                          `json:"removable"`
	Virtual           bool                          `json:"virtual"`
	Model             string                        `json:"model,omitempty"`
	Serial            string                        `json:"serial,omitempty"`
	Transport         string                        `json:"transport,omitempty"`
	Filesystem        *DiskFilesystem               `json:"filesystem,omitempty"`
	Mounts            []DiskMount                   `json:"mounts"`
	Protected         bool                          `json:"protected"`
	ProtectionReasons []string                      `json:"protectionReasons"`
	Operations        map[string]DiskOperationState `json:"operations"`
}

type DiskPartitionJob struct {
	ID           string     `json:"id"`
	Action       string     `json:"action"`
	DeviceID     string     `json:"deviceId"`
	DevicePath   string     `json:"devicePath"`
	Status       string     `json:"status"`
	Stage        string     `json:"stage"`
	Progress     int        `json:"progress"`
	Message      string     `json:"message"`
	RecoveryPath string     `json:"recoveryPath,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
	FinishedAt   *time.Time `json:"finishedAt,omitempty"`
}

type DiskPartitionSnapshot struct {
	ResourceVersion string            `json:"resourceVersion"`
	Platform        DiskPlatform      `json:"platform"`
	Devices         []DiskDevice      `json:"devices"`
	Job             *DiskPartitionJob `json:"job,omitempty"`
	ObservedAt      time.Time         `json:"observedAt"`
}

// DiskPartitionActionRequest is the only accepted disk mutation envelope.
// Pointer booleans preserve the distinction between an omitted field and an
// explicitly supplied false value.
type DiskPartitionActionRequest struct {
	Action                  string `json:"action"`
	DeviceID                string `json:"deviceId"`
	ExpectedResourceVersion string `json:"expectedResourceVersion"`
	MountPoint              string `json:"mountPoint,omitempty"`
	Persist                 *bool  `json:"persist,omitempty"`
	RemovePersistence       *bool  `json:"removePersistence,omitempty"`
	Filesystem              string `json:"filesystem,omitempty"`

	providedFields map[string]struct{}
}

type diskPartitionActionRequestJSON struct {
	Action                  string `json:"action"`
	DeviceID                string `json:"deviceId"`
	ExpectedResourceVersion string `json:"expectedResourceVersion"`
	MountPoint              string `json:"mountPoint,omitempty"`
	Persist                 *bool  `json:"persist,omitempty"`
	RemovePersistence       *bool  `json:"removePersistence,omitempty"`
	Filesystem              string `json:"filesystem,omitempty"`
}

// UnmarshalJSON rejects unknown fields, duplicate top-level fields and
// trailing JSON values. It also records exactly which fields were supplied so
// validation can reject action-inapplicable zero and null values.
func (request *DiskPartitionActionRequest) UnmarshalJSON(data []byte) error {
	if !utf8.Valid(data) {
		return fmt.Errorf("disk partition action request must be valid UTF-8")
	}
	fields, err := diskPartitionActionJSONFields(data)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var value diskPartitionActionRequestJSON
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple JSON values")
		}
		return err
	}

	*request = DiskPartitionActionRequest{
		Action:                  value.Action,
		DeviceID:                value.DeviceID,
		ExpectedResourceVersion: value.ExpectedResourceVersion,
		MountPoint:              value.MountPoint,
		Persist:                 value.Persist,
		RemovePersistence:       value.RemovePersistence,
		Filesystem:              value.Filesystem,
		providedFields:          fields,
	}
	return nil
}

// ValidateDiskPartitionAction returns a stable field/detail pair suitable for
// either a Panel or Agent 422 response.
func ValidateDiskPartitionAction(request *DiskPartitionActionRequest) (string, string) {
	if request == nil {
		return "request", "request is required"
	}
	if request.Action == "" {
		return "action", "action is required"
	}

	allowed := map[string]bool{
		"action":                  true,
		"deviceId":                true,
		"expectedResourceVersion": true,
	}
	switch request.Action {
	case DiskPartitionActionMount:
		allowed["mountPoint"] = true
		allowed["persist"] = true
	case DiskPartitionActionUnmount:
		allowed["mountPoint"] = true
		allowed["removePersistence"] = true
	case DiskPartitionActionFormat:
		allowed["filesystem"] = true
	case DiskPartitionActionCheck, DiskPartitionActionRepair:
	default:
		return "action", "unsupported disk partition action"
	}
	for field := range request.actualProvidedFields() {
		if !allowed[field] {
			return field, "field is not allowed for this action"
		}
	}

	if !diskOpaqueIDPattern.MatchString(request.DeviceID) {
		return "deviceId", "deviceId must be 64 lowercase hexadecimal characters"
	}
	if !diskOpaqueIDPattern.MatchString(request.ExpectedResourceVersion) {
		return "expectedResourceVersion", "expectedResourceVersion must be 64 lowercase hexadecimal characters"
	}

	switch request.Action {
	case DiskPartitionActionMount:
		if field, detail := validateDiskMountPoint(request.MountPoint); field != "" {
			return field, detail
		}
		if request.fieldProvided("persist") && request.Persist == nil {
			return "persist", "persist must be a boolean"
		}
	case DiskPartitionActionUnmount:
		if field, detail := validateDiskMountPoint(request.MountPoint); field != "" {
			return field, detail
		}
		if request.fieldProvided("removePersistence") && request.RemovePersistence == nil {
			return "removePersistence", "removePersistence must be a boolean"
		}
	case DiskPartitionActionFormat:
		switch request.Filesystem {
		case DiskFilesystemExt4, DiskFilesystemXFS, DiskFilesystemNTFS, DiskFilesystemVFAT:
		default:
			return "filesystem", "filesystem must be one of ext4, xfs, ntfs, or vfat"
		}
	}
	return "", ""
}

func (request *DiskPartitionActionRequest) actualProvidedFields() map[string]struct{} {
	if request.providedFields != nil {
		return request.providedFields
	}
	fields := map[string]struct{}{
		"action":                  {},
		"deviceId":                {},
		"expectedResourceVersion": {},
	}
	if request.MountPoint != "" {
		fields["mountPoint"] = struct{}{}
	}
	if request.Persist != nil {
		fields["persist"] = struct{}{}
	}
	if request.RemovePersistence != nil {
		fields["removePersistence"] = struct{}{}
	}
	if request.Filesystem != "" {
		fields["filesystem"] = struct{}{}
	}
	return fields
}

func (request *DiskPartitionActionRequest) fieldProvided(field string) bool {
	_, ok := request.actualProvidedFields()[field]
	return ok
}

func validateDiskMountPoint(value string) (string, string) {
	if value == "" {
		return "mountPoint", "mountPoint is required"
	}
	if len(value) > DiskMountPointMaxBytes {
		return "mountPoint", "mountPoint must be no longer than 4096 bytes"
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return "mountPoint", "mountPoint must not contain control characters"
		}
	}
	if !path.IsAbs(value) || path.Clean(value) != value {
		return "mountPoint", "mountPoint must be a canonical absolute Linux path"
	}
	return "", ""
}

func diskPartitionActionJSONFields(data []byte) (map[string]struct{}, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	opening, ok := token.(json.Delim)
	if !ok || opening != '{' {
		return nil, fmt.Errorf("disk partition action request must be a JSON object")
	}

	fields := make(map[string]struct{})
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		field, ok := token.(string)
		if !ok {
			return nil, fmt.Errorf("disk partition action field name must be a string")
		}
		if _, exists := fields[field]; exists {
			return nil, fmt.Errorf("duplicate JSON field %q", field)
		}
		fields[field] = struct{}{}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
	}
	closing, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if delimiter, ok := closing.(json.Delim); !ok || delimiter != '}' {
		return nil, fmt.Errorf("disk partition action request must be a JSON object")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("multiple JSON values")
		}
		return nil, err
	}
	return fields, nil
}
