package systemmanage

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kejilion/kejilion-panel/internal/contract"
)

func TestCollectDiskInventoryAcceptsNullLsblkFieldsAndBuildsOpaqueTopology(t *testing.T) {
	runner := &fakeRunner{}
	manager, _, _, _ := testManager(t, runner)
	runner.run = func(_ context.Context, name string, _ ...string) ([]byte, error) {
		if name != "lsblk" {
			return nil, nil
		}
		return []byte(`{"blockdevices":[{"name":"/dev/sdb","kname":"/dev/sdb","path":"/dev/sdb","type":"disk","pkname":null,"size":1073741824,"ro":0,"rm":1,"model":null,"serial":"disk-serial","tran":"usb","wwn":null,"fstype":null,"fsver":null,"label":null,"uuid":null,"partuuid":null,"maj:min":"8:16","mountpoints":[null],"children":[{"name":"/dev/sdb1","kname":"/dev/sdb1","path":"/dev/sdb1","type":"part","pkname":"/dev/sdb","size":"1072693248","ro":"0","rm":"1","model":null,"serial":null,"tran":null,"wwn":null,"fstype":"ext4","fsver":"1.0","label":null,"uuid":"12345678-1234-1234-1234-123456789abc","partuuid":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee","maj:min":"8:17","mountpoints":[]}]}]}`), nil
	}
	envelope, err := manager.collectDiskInventory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(envelope.Snapshot.Devices) != 2 || len(envelope.Targets) != 2 {
		t.Fatalf("unexpected inventory sizes: %#v", envelope)
	}
	if len(envelope.Snapshot.ResourceVersion) != 64 {
		t.Fatalf("resourceVersion = %q", envelope.Snapshot.ResourceVersion)
	}
	for _, device := range envelope.Snapshot.Devices {
		if len(device.ID) != 64 || strings.Contains(device.ID, "/dev/") {
			t.Fatalf("device ID is not opaque: %q", device.ID)
		}
	}
	parent := envelope.Snapshot.Devices[0]
	child := envelope.Snapshot.Devices[1]
	if parent.Path > child.Path {
		parent, child = child, parent
	}
	if child.ParentID != parent.ID {
		t.Fatalf("child parentId = %q, want %q", child.ParentID, parent.ID)
	}
	if !parent.Protected || !strings.Contains(strings.Join(parent.ProtectionReasons, " "), "子设备") {
		t.Fatalf("non-leaf disk was not protected: %#v", parent)
	}
}

func TestDiskResourceVersionExcludesUsageAndObservationData(t *testing.T) {
	totalA, usedA, totalB, usedB := uint64(100), uint64(25), uint64(1000), uint64(900)
	percentA, percentB := 25.0, 90.0
	device := contract.DiskDevice{ID: strings.Repeat("a", 64), Path: "/dev/sdb1", Name: "sdb1", Type: "part", SizeBytes: 1000,
		Mounts:            []contract.DiskMount{{Path: "/mnt/data", Persistent: true, TotalBytes: &totalA, UsedBytes: &usedA, UsagePercent: &percentA}},
		ProtectionReasons: []string{}, Operations: map[string]contract.DiskOperationState{}}
	target := diskTarget{ID: device.ID, Path: device.Path, MajorMinor: "8:17", PersistentMounts: []string{"/mnt/data"}}
	first := diskResourceVersion([]contract.DiskDevice{device}, []diskTarget{target})
	device.Mounts[0].TotalBytes, device.Mounts[0].UsedBytes, device.Mounts[0].UsagePercent = &totalB, &usedB, &percentB
	second := diskResourceVersion([]contract.DiskDevice{device}, []diskTarget{target})
	if first != second {
		t.Fatalf("usage changed resourceVersion: %s != %s", first, second)
	}
	target.PersistentMounts = nil
	if third := diskResourceVersion([]contract.DiskDevice{device}, []diskTarget{target}); third == second {
		t.Fatal("persistence change did not affect resourceVersion")
	}
}

func TestParseDiskReceiptIsStrictAndRetainsRollbackRecovery(t *testing.T) {
	receiptText := "KPANEL_DISK_MANAGEMENT_PROTOCOL 1\n" +
		"KPANEL_DISK_MANAGEMENT_STATUS=rollback-failed\n" +
		"KPANEL_DISK_MANAGEMENT_DEVICE=8:17\n" +
		"KPANEL_DISK_MANAGEMENT_MESSAGE_HEX=E99C80E8A681E4BABAE5B7A5E5A484E79086\n" +
		"KPANEL_DISK_MANAGEMENT_BACKUP_HEX=2f7661722f6c69622f6b70616e656c2f7265636f76657279\n"
	// Uppercase hex is deliberately rejected; the protocol is canonical.
	if _, err := parseDiskReceipt([]byte(receiptText), "8:17"); err == nil {
		t.Fatal("uppercase receipt hex was accepted")
	}
	receiptText = strings.ToLower(receiptText[strings.Index(receiptText, "KPANEL"):])
	// Restore protocol keys, keeping payload hex lowercase.
	receiptText = "KPANEL_DISK_MANAGEMENT_PROTOCOL 1\nKPANEL_DISK_MANAGEMENT_STATUS=rollback-failed\nKPANEL_DISK_MANAGEMENT_DEVICE=8:17\nKPANEL_DISK_MANAGEMENT_MESSAGE_HEX=e99c80e8a681e4babae5b7a5e5a484e79086\nKPANEL_DISK_MANAGEMENT_BACKUP_HEX=2f7661722f6c69622f6b70616e656c2f7265636f76657279\n"
	receipt, err := parseDiskReceipt([]byte(receiptText), "8:17")
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "rollback-failed" || receipt.Message == "" || receipt.BackupPath != "/var/lib/kpanel/recovery" {
		t.Fatalf("receipt recovery was lost: %#v", receipt)
	}
	for name, invalid := range map[string]string{
		"noise":       "noise\n" + receiptText,
		"truncated":   strings.Join(strings.Split(receiptText, "\n")[:4], "\n") + "\n",
		"wrong-order": strings.Replace(receiptText, "KPANEL_DISK_MANAGEMENT_STATUS=rollback-failed\nKPANEL_DISK_MANAGEMENT_DEVICE=8:17", "KPANEL_DISK_MANAGEMENT_DEVICE=8:17\nKPANEL_DISK_MANAGEMENT_STATUS=rollback-failed", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseDiskReceipt([]byte(invalid), "8:17"); err == nil {
				t.Fatal("invalid receipt accepted")
			}
		})
	}
}

func TestDiskMountDestinationRejectsCriticalOccupiedAndNonEmptyPaths(t *testing.T) {
	manager, _, procRoot, _ := testManager(t, &fakeRunner{})
	if err := os.MkdirAll(filepath.Join(procRoot, "self"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(procRoot, "self", "mountinfo"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	envelope := diskInspectEnvelope{}
	for _, mountPoint := range []string{"/", "/dev", "/proc/x", "/sys", "/run/kpanel", "/boot", "/var/lib/kejilion-panel/jobs", "/home", "/home/data", "/home/docker"} {
		if err := manager.validateDiskMountDestination(mountPoint, strings.Repeat("a", 64), envelope); err == nil {
			t.Fatalf("critical mount point %q was accepted", mountPoint)
		}
	}
	envelope.Snapshot.Devices = []contract.DiskDevice{{Mounts: []contract.DiskMount{{Path: "/mnt/occupied"}}}}
	if err := manager.validateDiskMountDestination("/mnt/occupied", strings.Repeat("a", 64), envelope); err == nil {
		t.Fatal("occupied mount point was accepted")
	}
	envelope.Snapshot.Devices = nil
	nonEmpty := t.TempDir()
	if err := os.WriteFile(filepath.Join(nonEmpty, "payload"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := manager.validateDiskMountDestination(nonEmpty, strings.Repeat("a", 64), envelope); err == nil {
		t.Fatal("non-empty directory was accepted")
	}
}

func TestReadBoundedDiskJSONRejectsTrailingGarbage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "job.json")
	if err := os.WriteFile(path, []byte(`{"id":"ok"}garbage`), 0600); err != nil {
		t.Fatal(err)
	}
	var value map[string]string
	if err := readBoundedDiskJSON(path, 1024, &value); err == nil {
		t.Fatal("trailing invalid JSON was accepted")
	}
}

func TestDiskActionUnitUsesOnlyTrustedDeviceAndRequiredMountPolicy(t *testing.T) {
	runner := &fakeRunner{}
	stateDir := t.TempDir()
	executable := filepath.Join(t.TempDir(), "kejilion-agent")
	manager := NewManager(Config{Enabled: true, StateDir: stateDir, Executable: executable, Runner: runner, EffectiveUID: func() int { return 0 }})
	record := diskJobRecord{ID: strings.Repeat("a", 32), Target: diskTarget{Path: "/dev/sdb1"}, Request: contract.DiskPartitionActionRequest{Action: "mount"}}
	if err := manager.launchDiskJob(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	command := strings.Join(runner.commands, "\n")
	for _, required := range []string{"--no-block", "PrivateDevices=no", "PrivateMounts=no", "DevicePolicy=closed", "DeviceAllow=/dev/sdb1 rw", "CAP_SYS_ADMIN CAP_DAC_OVERRIDE CAP_FOWNER", "SystemCallFilter=@system-service @mount", "NoNewPrivileges=yes", "TimeoutStartSec=2min", "disk-run --state-dir " + stateDir + " --id " + record.ID} {
		if !strings.Contains(command, required) {
			t.Fatalf("action unit omitted %q:\n%s", required, command)
		}
	}
	for _, forbidden := range []string{"PrivateTmp=", "ProtectSystem=", "ProtectHome="} {
		if strings.Contains(command, forbidden) {
			t.Fatalf("action unit added mount-isolating property %q", forbidden)
		}
	}
}

func TestDiskProtectedMountPolicyMatchesScriptCriticalPrefixes(t *testing.T) {
	for _, value := range []string{"/", "/boot/efi", "/home/data", "/var/lib/kejilion-panel/system", "/run/media"} {
		if !diskProtectedMountPoint(value) {
			t.Fatalf("critical mount %q was not protected", value)
		}
	}
	for _, value := range []string{"/mnt/data", "/media/archive", "/srv/storage"} {
		if diskProtectedMountPoint(value) {
			t.Fatalf("ordinary data mount %q was over-protected", value)
		}
	}
}

func TestDiskPlatformTreatsSystemdWSLMarkerAsWSLNotContainer(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux platform detection")
	}
	manager, _, procRoot, _ := testManager(t, &fakeRunner{})
	manager.runRoot = filepath.Join(t.TempDir(), "run")
	for _, directory := range []string{filepath.Join(procRoot, "sys", "kernel"), filepath.Join(procRoot, "1"), filepath.Join(manager.runRoot, "systemd")} {
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(procRoot, "sys", "kernel", "osrelease"), []byte("6.6.87.2-microsoft-standard-WSL2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(procRoot, "1", "cgroup"), []byte("0::/init.scope\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manager.runRoot, "systemd", "container"), []byte("wsl\n"), 0600); err != nil {
		t.Fatal(err)
	}
	platform := manager.diskPlatform()
	if platform.Kind != "wsl2" || !platform.Writable {
		t.Fatalf("WSL marker misclassified: %#v", platform)
	}
}

func TestOpaqueDiskIDsDisambiguateInheritedSerialAndWWN(t *testing.T) {
	parent := diskInventoryNode{lsblkNode: lsblkNode{Type: "disk", Serial: "same", WWN: "same", MajorMinor: "8:16"}}
	child := diskInventoryNode{lsblkNode: lsblkNode{Type: "part", Serial: "same", WWN: "same", MajorMinor: "8:17"}}
	parent.id, child.id = opaqueDiskID(parent.lsblkNode), opaqueDiskID(child.lsblkNode)
	nodes := []diskInventoryNode{parent, child}
	disambiguateDiskIDs(nodes)
	if nodes[0].id == nodes[1].id || len(nodes[0].id) != 64 || len(nodes[1].id) != 64 {
		t.Fatalf("inherited stable facts collided: %#v", nodes)
	}
}

func TestDiskFilesystemToolsDisableUnavailableOperation(t *testing.T) {
	runner := &fakeRunner{missing: map[string]bool{"mount": true, "e2fsck": true}}
	manager := NewManager(Config{Runner: runner})
	device := contract.DiskDevice{Filesystem: &contract.DiskFilesystem{Type: "ext4"}, Mounts: []contract.DiskMount{}, ProtectionReasons: []string{}}
	operations := manager.diskOperations(device, "", false)
	if operations["mount"].Enabled || !strings.Contains(operations["mount"].Reason, "mount") {
		t.Fatalf("mount tool absence not reflected: %#v", operations["mount"])
	}
	if operations["check"].Enabled || operations["repair"].Enabled {
		t.Fatalf("filesystem check tool absence not reflected: %#v", operations)
	}
	request := contract.DiskPartitionActionRequest{Action: "format", Filesystem: "ext4"}
	runner.missing["mkfs.ext4"] = true
	if err := manager.validateDiskActionTarget(request, device, diskTarget{}, diskInspectEnvelope{}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("missing selected formatter error = %v", err)
	}
}

func TestDiskPersistentMountsResolveUUIDAndPartUUID(t *testing.T) {
	manager, etcRoot, _, _ := testManager(t, &fakeRunner{})
	if err := os.WriteFile(filepath.Join(etcRoot, "fstab"), []byte("UUID=fs-uuid /mnt/data ext4 defaults 0 2\nPARTUUID=part-uuid /mnt/part ext4 defaults 0 2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	nodes := []diskInventoryNode{{lsblkNode: lsblkNode{Path: "/dev/sdb1", UUID: "fs-uuid", PartUUID: "part-uuid"}}}
	live, byDevice := manager.diskPersistentMounts(nodes)
	if !live["/dev/sdb1\x00/mnt/data"] || !live["/dev/sdb1\x00/mnt/part"] || len(byDevice["/dev/sdb1"]) != 2 {
		t.Fatalf("persistent UUID entries were not resolved: %#v %#v", live, byDevice)
	}
}

func TestDiskPersistentMountsResolveDevSymlinkWithoutExecutingSource(t *testing.T) {
	manager, etcRoot, _, _ := testManager(t, &fakeRunner{})
	if err := os.WriteFile(filepath.Join(etcRoot, "fstab"), []byte("/dev/disk/by-uuid/fs /mnt/data ext4 defaults 0 2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	original := diskEvalSymlinks
	diskEvalSymlinks = func(value string) (string, error) {
		if value == "/dev/disk/by-uuid/fs" {
			return "/dev/sdb1", nil
		}
		return value, nil
	}
	defer func() { diskEvalSymlinks = original }()
	nodes := []diskInventoryNode{{lsblkNode: lsblkNode{Path: "/dev/sdb1"}}}
	live, byDevice := manager.diskPersistentMounts(nodes)
	if !live["/dev/sdb1\x00/mnt/data"] || len(byDevice["/dev/sdb1"]) != 1 {
		t.Fatalf("/dev symlink was not mapped to the trusted target: %#v %#v", live, byDevice)
	}
}

func TestDiskMountUnknownUsageIsOmittedFromJSON(t *testing.T) {
	mount := contract.DiskMount{Path: "/unreadable", Persistent: false}
	data, err := json.Marshal(mount)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"totalBytes", "usedBytes", "availableBytes", "usagePercent"} {
		if strings.Contains(string(data), field) {
			t.Fatalf("unknown usage field %q was serialized: %s", field, data)
		}
	}
}

func TestDiskPartitionsUsesReadOnlyTransientInspectUnit(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("transient systemd policy is Linux-only")
	}
	runner := &fakeRunner{}
	executable := filepath.Join(t.TempDir(), "kejilion-agent")
	id := strings.Repeat("a", 64)
	version := strings.Repeat("b", 64)
	runner.run = func(_ context.Context, name string, _ ...string) ([]byte, error) {
		if name != "systemd-run" {
			return nil, nil
		}
		return []byte(`{"snapshot":{"resourceVersion":"` + version + `","platform":{"kind":"linux","label":"Linux","writable":true},"devices":[{"id":"` + id + `","path":"/dev/sdb","name":"sdb","type":"disk","sizeBytes":1,"readOnly":false,"removable":false,"virtual":false,"mounts":[],"protected":false,"protectionReasons":[],"operations":{}}],"observedAt":"2026-08-23T00:00:00Z"},"targets":[{"id":"` + id + `","path":"/dev/sdb","majorMinor":"8:16","kname":"sdb","persistentMounts":[]}]}`), nil
	}
	manager := NewManager(Config{Enabled: true, StateDir: t.TempDir(), Executable: executable, Runner: runner, EffectiveUID: func() int { return 0 }})
	if _, err := manager.DiskPartitions(context.Background()); err != nil {
		t.Fatal(err)
	}
	command := strings.Join(runner.commands, "\n")
	for _, required := range []string{"--wait", "--pipe", "--collect", "PrivateDevices=no", "PrivateMounts=no", "NoNewPrivileges=yes", "CapabilityBoundingSet=", "AmbientCapabilities=", "RestrictNamespaces=yes", "disk-inspect --state-dir"} {
		if !strings.Contains(command, required) {
			t.Fatalf("inspect unit omitted %q:\n%s", required, command)
		}
	}
	for _, forbidden := range []string{"PrivateTmp=", "ProtectSystem=", "DeviceAllow="} {
		if strings.Contains(command, forbidden) {
			t.Fatalf("inspect unit added forbidden property %q", forbidden)
		}
	}
}

func TestDiskScriptTrustRejectsWritableAndSymlinkedProtocol(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("root ownership trust is Linux-only")
	}
	script := filepath.Join(t.TempDir(), "kejilion.sh")
	content := "#!/bin/bash\nKPANEL_DISK_MANAGEMENT_PROTOCOL_VERSION=\"1\"\nKJ_DISK_MANAGEMENT_NONINTERACTIVE=1\nkpanel_disk_management_dispatch() { :; }\n"
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{}
	manager := NewManager(Config{Enabled: true, Executable: filepath.Join(t.TempDir(), "agent"), Runner: runner, EffectiveUID: func() int { return 0 }, DiskScript: func() (string, error) { return script, nil }})
	if err := manager.diskWriteAvailability(); err != nil {
		t.Fatalf("trusted protocol rejected: %v", err)
	}
	if err := os.Chmod(script, 0775); err != nil {
		t.Fatal(err)
	}
	if err := manager.diskWriteAvailability(); err == nil {
		t.Fatal("group-writable script was trusted")
	}
	if err := os.Chmod(script, 0755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "kejilion.sh")
	if err := os.Symlink(script, link); err != nil {
		t.Fatal(err)
	}
	manager.diskScript = func() (string, error) { return link, nil }
	if err := manager.diskWriteAvailability(); err == nil {
		t.Fatal("symlinked script was trusted")
	}
}

func TestStartDiskPartitionActionConflictsAndPersistsBeforeLaunch(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("durable disk jobs are Linux-only")
	}
	for _, scenario := range []string{"stale-version", "active-job", "persist-before-launch"} {
		t.Run(scenario, func(t *testing.T) {
			runner := &fakeRunner{}
			manager, _, _, stateDir := testManager(t, runner)
			manager.diskScript = trustedDiskTestScript(t)
			manager.executable = filepath.Join(t.TempDir(), "agent")
			deviceID, version := strings.Repeat("a", 64), strings.Repeat("b", 64)
			envelope := diskInspectEnvelope{Snapshot: contract.DiskPartitionSnapshot{
				ResourceVersion: version, Platform: contract.DiskPlatform{Kind: "linux", Label: "Linux", Writable: true},
				Devices:    []contract.DiskDevice{{ID: deviceID, Path: "/dev/sde", Name: "sde", Type: "disk", SizeBytes: 1024, Filesystem: &contract.DiskFilesystem{Type: "ext4"}, Mounts: []contract.DiskMount{}, ProtectionReasons: []string{}, Operations: map[string]contract.DiskOperationState{"check": {Enabled: true}}}},
				ObservedAt: manager.now().UTC(),
			}, Targets: []diskTarget{{ID: deviceID, Path: "/dev/sde", MajorMinor: "8:64", KName: "sde", PersistentMounts: []string{}, Holders: []string{}}}}
			inspectJSON, err := json.Marshal(envelope)
			if err != nil {
				t.Fatal(err)
			}
			launches := 0
			runner.run = func(_ context.Context, name string, arguments ...string) ([]byte, error) {
				joined := strings.Join(arguments, " ")
				if name == "systemd-run" && strings.Contains(joined, "disk-inspect") {
					return inspectJSON, nil
				}
				if name == "systemd-run" && strings.Contains(joined, "disk-run") {
					launches++
					if scenario == "persist-before-launch" {
						record, recordErr := manager.readDiskRecord()
						job := manager.readDiskJob()
						if recordErr != nil || record.ID == "" || job == nil || job.Status != contract.DiskPartitionJobQueued {
							t.Fatalf("launch preceded durable state: record=%#v err=%v job=%#v", record, recordErr, job)
						}
					}
				}
				return nil, nil
			}
			request := contract.DiskPartitionActionRequest{Action: contract.DiskPartitionActionCheck, DeviceID: deviceID, ExpectedResourceVersion: version}
			if scenario == "stale-version" {
				request.ExpectedResourceVersion = strings.Repeat("c", 64)
			}
			if scenario == "active-job" {
				if err := manager.ensureDiskJobDir(); err != nil {
					t.Fatal(err)
				}
				active := contract.DiskPartitionJob{ID: strings.Repeat("d", 32), Action: "check", DeviceID: deviceID, DevicePath: "/dev/sde", Status: contract.DiskPartitionJobRunning, Stage: "executing", Progress: 30, Message: "running", CreatedAt: manager.now().UTC()}
				if err := manager.writeDiskJob(active); err != nil {
					t.Fatal(err)
				}
			}
			job, err := manager.StartDiskPartitionAction(context.Background(), request)
			switch scenario {
			case "persist-before-launch":
				if err != nil || job.Status != contract.DiskPartitionJobQueued || launches != 1 {
					t.Fatalf("job=%#v launches=%d err=%v state=%s", job, launches, err, stateDir)
				}
			default:
				if !errors.Is(err, ErrConflict) || launches != 0 {
					t.Fatalf("expected conflict before launch: launches=%d err=%v", launches, err)
				}
			}
		})
	}
}

func TestRunDiskJobVerifiesRealStateAndPersistsTerminalReceipt(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("durable disk jobs are Linux-only")
	}
	for _, scriptStatus := range []string{"applied", "rollback-failed"} {
		t.Run(scriptStatus, func(t *testing.T) {
			runner := &fakeRunner{}
			manager, _, _, _ := testManager(t, runner)
			manager.diskScript = trustedDiskTestScript(t)
			manager.executable = filepath.Join(t.TempDir(), "agent")
			lsblk := []byte(`{"blockdevices":[{"name":"/dev/sde","kname":"/dev/sde","path":"/dev/sde","type":"disk","pkname":null,"size":1073741824,"ro":0,"rm":0,"model":"Disk","serial":"disk-sde","tran":null,"wwn":"wwn-sde","fstype":"ext4","fsver":"1.0","label":null,"uuid":"12345678-1234-1234-1234-123456789abc","partuuid":null,"maj:min":"8:64","mountpoints":[]}]}`)
			message := hex.EncodeToString([]byte("structured script result"))
			backup := ""
			if scriptStatus == "rollback-failed" {
				backup = hex.EncodeToString([]byte("/var/lib/kejilion-panel/system/recovery/disk"))
			}
			scriptReceipt := []byte("KPANEL_DISK_MANAGEMENT_PROTOCOL 1\nKPANEL_DISK_MANAGEMENT_STATUS=" + scriptStatus + "\nKPANEL_DISK_MANAGEMENT_DEVICE=8:64\nKPANEL_DISK_MANAGEMENT_MESSAGE_HEX=" + message + "\nKPANEL_DISK_MANAGEMENT_BACKUP_HEX=" + backup + "\n")
			runner.run = func(_ context.Context, name string, _ ...string) ([]byte, error) {
				switch name {
				case "lsblk":
					return lsblk, nil
				case "env":
					if scriptStatus == "rollback-failed" {
						return scriptReceipt, errors.New("exit status 1")
					}
					return scriptReceipt, nil
				default:
					return nil, nil
				}
			}
			inventory, err := manager.collectDiskInventory(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			target, device, ok := findDiskTarget(inventory, inventory.Snapshot.Devices[0].ID)
			if !ok {
				t.Fatal("test target missing")
			}
			jobID := strings.Repeat("e", 32)
			request := contract.DiskPartitionActionRequest{Action: contract.DiskPartitionActionCheck, DeviceID: device.ID, ExpectedResourceVersion: inventory.Snapshot.ResourceVersion}
			record := diskJobRecord{ID: jobID, Request: request, Target: target, ExpectedResourceVersion: inventory.Snapshot.ResourceVersion}
			if err := manager.ensureDiskJobDir(); err != nil {
				t.Fatal(err)
			}
			if err := manager.writeDiskRecord(record); err != nil {
				t.Fatal(err)
			}
			if err := manager.writeDiskJob(contract.DiskPartitionJob{ID: jobID, Action: "check", DeviceID: device.ID, DevicePath: device.Path, Status: contract.DiskPartitionJobQueued, Stage: "launching", Progress: 2, Message: "queued", CreatedAt: manager.now().UTC()}); err != nil {
				t.Fatal(err)
			}
			runErr := manager.RunDiskJob(context.Background(), jobID)
			job := manager.readDiskJob()
			receipt, receiptErr := manager.readDiskReceipt()
			if receiptErr != nil || job == nil {
				t.Fatalf("terminal state missing: job=%#v receipt=%#v err=%v", job, receipt, receiptErr)
			}
			if scriptStatus == "applied" {
				if runErr != nil || job.Status != contract.DiskPartitionJobSucceeded || receipt.Status != contract.DiskPartitionJobSucceeded {
					t.Fatalf("verified success lost: job=%#v receipt=%#v err=%v", job, receipt, runErr)
				}
			} else {
				if runErr == nil || job.Status != contract.DiskPartitionJobNeedsAttention || receipt.Status != contract.DiskPartitionJobNeedsAttention || job.RecoveryPath == "" || receipt.BackupPath != job.RecoveryPath {
					t.Fatalf("rollback recovery lost: job=%#v receipt=%#v err=%v", job, receipt, runErr)
				}
			}
		})
	}
}

func TestDiskJobReconcileNeverTreatsSystemdExitZeroAsSuccess(t *testing.T) {
	runner := &fakeRunner{run: func(_ context.Context, name string, _ ...string) ([]byte, error) {
		if name == "systemctl" {
			return []byte("LoadState=not-found\nActiveState=inactive\nSubState=dead\nResult=success\nExecMainStatus=0\n"), nil
		}
		return nil, nil
	}}
	manager, _, _, _ := testManager(t, runner)
	if err := manager.ensureDiskJobDir(); err != nil {
		t.Fatal(err)
	}
	job := contract.DiskPartitionJob{ID: strings.Repeat("f", 32), Action: "check", DeviceID: strings.Repeat("a", 64), DevicePath: "/dev/sde", Status: contract.DiskPartitionJobRunning, Stage: "executing", Progress: 30, Message: "running", CreatedAt: manager.now().Add(-3 * time.Minute)}
	if err := manager.writeDiskJob(job); err != nil {
		t.Fatal(err)
	}
	manager.reconcileDiskJob(&job)
	if job.Status != contract.DiskPartitionJobNeedsAttention || job.Stage != "completion_unverified" {
		t.Fatalf("collected unit was misreported: %#v", job)
	}
}

func trustedDiskTestScript(t *testing.T) KejilionScriptFinder {
	t.Helper()
	path := filepath.Join(t.TempDir(), "kejilion.sh")
	content := "#!/bin/bash\nKPANEL_DISK_MANAGEMENT_PROTOCOL_VERSION=\"1\"\nKJ_DISK_MANAGEMENT_NONINTERACTIVE=1\nkpanel_disk_management_dispatch() { :; }\n"
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
	return func() (string, error) { return path, nil }
}
