package contract

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDiskPartitionActionStrictJSONAndFieldTracking(t *testing.T) {
	id := strings.Repeat("a", 64)
	version := strings.Repeat("b", 64)
	tests := []struct {
		name      string
		body      string
		wantError bool
		field     string
	}{
		{
			name:      "unknown field",
			body:      `{"action":"check","deviceId":"` + id + `","expectedResourceVersion":"` + version + `","unknown":true}`,
			wantError: true,
		},
		{
			name:      "duplicate field",
			body:      `{"action":"check","action":"repair","deviceId":"` + id + `","expectedResourceVersion":"` + version + `"}`,
			wantError: true,
		},
		{
			name:      "multiple values",
			body:      `{"action":"check"} {}`,
			wantError: true,
		},
		{
			name:      "non object",
			body:      `null`,
			wantError: true,
		},
		{
			name:  "inapplicable false field",
			body:  `{"action":"check","deviceId":"` + id + `","expectedResourceVersion":"` + version + `","persist":false}`,
			field: "persist",
		},
		{
			name:  "inapplicable empty field",
			body:  `{"action":"repair","deviceId":"` + id + `","expectedResourceVersion":"` + version + `","mountPoint":""}`,
			field: "mountPoint",
		},
		{
			name:  "null optional boolean",
			body:  `{"action":"mount","deviceId":"` + id + `","expectedResourceVersion":"` + version + `","mountPoint":"/mnt/data","persist":null}`,
			field: "persist",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var request DiskPartitionActionRequest
			err := json.Unmarshal([]byte(test.body), &request)
			if test.wantError {
				if err == nil {
					t.Fatal("JSON was accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			field, detail := ValidateDiskPartitionAction(&request)
			if field != test.field {
				t.Fatalf("field=%q detail=%q want=%q", field, detail, test.field)
			}
		})
	}
}

func TestValidateDiskPartitionAction(t *testing.T) {
	id := strings.Repeat("a", 64)
	version := strings.Repeat("b", 64)
	falseValue := false
	trueValue := true
	tests := []struct {
		name    string
		request DiskPartitionActionRequest
		field   string
	}{
		{
			name: "mount",
			request: DiskPartitionActionRequest{
				Action: DiskPartitionActionMount, DeviceID: id, ExpectedResourceVersion: version,
				MountPoint: "/mnt/data", Persist: &falseValue,
			},
		},
		{
			name: "unmount",
			request: DiskPartitionActionRequest{
				Action: DiskPartitionActionUnmount, DeviceID: id, ExpectedResourceVersion: version,
				MountPoint: "/srv/storage", RemovePersistence: &trueValue,
			},
		},
		{
			name: "format each supported filesystem",
			request: DiskPartitionActionRequest{
				Action: DiskPartitionActionFormat, DeviceID: id, ExpectedResourceVersion: version,
				Filesystem: DiskFilesystemVFAT,
			},
		},
		{
			name: "check",
			request: DiskPartitionActionRequest{
				Action: DiskPartitionActionCheck, DeviceID: id, ExpectedResourceVersion: version,
			},
		},
		{
			name: "repair",
			request: DiskPartitionActionRequest{
				Action: DiskPartitionActionRepair, DeviceID: id, ExpectedResourceVersion: version,
			},
		},
		{
			name: "unsupported action",
			request: DiskPartitionActionRequest{
				Action: "resize", DeviceID: id, ExpectedResourceVersion: version,
			},
			field: "action",
		},
		{
			name: "uppercase device ID",
			request: DiskPartitionActionRequest{
				Action: DiskPartitionActionCheck, DeviceID: strings.Repeat("A", 64), ExpectedResourceVersion: version,
			},
			field: "deviceId",
		},
		{
			name: "short device ID",
			request: DiskPartitionActionRequest{
				Action: DiskPartitionActionCheck, DeviceID: strings.Repeat("a", 63), ExpectedResourceVersion: version,
			},
			field: "deviceId",
		},
		{
			name: "uppercase resource version",
			request: DiskPartitionActionRequest{
				Action: DiskPartitionActionCheck, DeviceID: id, ExpectedResourceVersion: strings.Repeat("B", 64),
			},
			field: "expectedResourceVersion",
		},
		{
			name: "missing mount point",
			request: DiskPartitionActionRequest{
				Action: DiskPartitionActionMount, DeviceID: id, ExpectedResourceVersion: version,
			},
			field: "mountPoint",
		},
		{
			name: "relative mount point",
			request: DiskPartitionActionRequest{
				Action: DiskPartitionActionMount, DeviceID: id, ExpectedResourceVersion: version,
				MountPoint: "mnt/data",
			},
			field: "mountPoint",
		},
		{
			name: "unclean mount point",
			request: DiskPartitionActionRequest{
				Action: DiskPartitionActionUnmount, DeviceID: id, ExpectedResourceVersion: version,
				MountPoint: "/mnt/../data",
			},
			field: "mountPoint",
		},
		{
			name: "trailing slash",
			request: DiskPartitionActionRequest{
				Action: DiskPartitionActionMount, DeviceID: id, ExpectedResourceVersion: version,
				MountPoint: "/mnt/data/",
			},
			field: "mountPoint",
		},
		{
			name: "control character",
			request: DiskPartitionActionRequest{
				Action: DiskPartitionActionMount, DeviceID: id, ExpectedResourceVersion: version,
				MountPoint: "/mnt/\u0085data",
			},
			field: "mountPoint",
		},
		{
			name: "oversized mount point",
			request: DiskPartitionActionRequest{
				Action: DiskPartitionActionMount, DeviceID: id, ExpectedResourceVersion: version,
				MountPoint: "/" + strings.Repeat("x", DiskMountPointMaxBytes),
			},
			field: "mountPoint",
		},
		{
			name: "unsupported filesystem",
			request: DiskPartitionActionRequest{
				Action: DiskPartitionActionFormat, DeviceID: id, ExpectedResourceVersion: version,
				Filesystem: "btrfs",
			},
			field: "filesystem",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			field, detail := ValidateDiskPartitionAction(&test.request)
			if field != test.field {
				t.Fatalf("field=%q detail=%q want=%q", field, detail, test.field)
			}
		})
	}
}

func TestValidateDiskPartitionFormatFilesystems(t *testing.T) {
	id := strings.Repeat("a", 64)
	version := strings.Repeat("b", 64)
	for _, filesystem := range []string{
		DiskFilesystemExt4,
		DiskFilesystemXFS,
		DiskFilesystemNTFS,
		DiskFilesystemVFAT,
	} {
		t.Run(filesystem, func(t *testing.T) {
			request := DiskPartitionActionRequest{
				Action: DiskPartitionActionFormat, DeviceID: id,
				ExpectedResourceVersion: version, Filesystem: filesystem,
			}
			if field, detail := ValidateDiskPartitionAction(&request); field != "" {
				t.Fatalf("field=%q detail=%q", field, detail)
			}
		})
	}
}

func TestDiskPartitionActionRejectsInvalidUTF8(t *testing.T) {
	data := append([]byte(`{"action":"mount","mountPoint":"/mnt/`), 0xff)
	data = append(data, []byte(`"}`)...)
	var request DiskPartitionActionRequest
	if err := json.Unmarshal(data, &request); err == nil {
		t.Fatal("invalid UTF-8 request was accepted")
	}
}
