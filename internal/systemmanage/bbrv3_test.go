package systemmanage

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseBBRv3Status(t *testing.T) {
	status, err := parseBBRv3Status([]byte(
		"KPANEL_BBRV3_PROTOCOL 1\n" +
			`KPANEL_BBRV3_STATUS {"supported":true,"installed":true,"active":false,"architecture":"x86_64","os":"debian","codename":"bookworm","runningKernel":"6.1.0-amd64","installedKernel":"6.12.0-x64v3-xanmod1","congestionControl":"bbr","defaultQDisc":"fq","rebootRequired":true,"reason":""}` +
			"\n",
	))
	if err != nil {
		t.Fatal(err)
	}
	if !status.Supported || !status.Installed || status.Active ||
		status.Architecture != "x86_64" || status.Codename != "bookworm" ||
		!status.RebootRequired {
		t.Fatalf("BBRv3 status = %#v", status)
	}

	for name, output := range map[string]string{
		"missing marker": `{"supported":true}`,
		"unknown field":  `KPANEL_BBRV3_STATUS {"supported":true,"extra":true}`,
		"unknown reason": `KPANEL_BBRV3_STATUS {"reason":"arbitrary"}`,
		"trailing JSON":  `KPANEL_BBRV3_STATUS {"supported":true} {}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseBBRv3Status([]byte(output)); err == nil {
				t.Fatalf("invalid status accepted: %q", output)
			}
		})
	}
}

func TestVerifyBBRv3ActionOutputRequiresReceiptAndArtifacts(t *testing.T) {
	validInstall := []byte(
		`KPANEL_BBRV3_RESULT {"action":"install","changed":true,"rebootRequired":true}` + "\n" +
			`KPANEL_BBRV3_STATUS {"supported":true,"installed":true,"active":false,"architecture":"x86_64","os":"debian","codename":"bookworm","runningKernel":"6.1.0-amd64","installedKernel":"6.12.0-x64v3-xanmod1","congestionControl":"bbr","defaultQDisc":"fq","rebootRequired":true,"reason":""}` + "\n",
	)
	rebootRequired, err := verifyBBRv3ActionOutput(validInstall, "install")
	if err != nil {
		t.Fatal(err)
	}
	if !rebootRequired {
		t.Fatal("verified BBRv3 install did not preserve reboot requirement")
	}
	for name, output := range map[string][]byte{
		"missing receipt": []byte(
			`KPANEL_BBRV3_STATUS {"supported":true,"installed":true,"reason":""}`,
		),
		"wrong action": []byte(
			`KPANEL_BBRV3_RESULT {"action":"update","changed":true,"rebootRequired":true}` + "\n" +
				`KPANEL_BBRV3_STATUS {"supported":true,"installed":true,"reason":""}`,
		),
		"missing package": []byte(
			`KPANEL_BBRV3_RESULT {"action":"install","changed":true,"rebootRequired":true}` + "\n" +
				`KPANEL_BBRV3_STATUS {"supported":true,"installed":false,"reason":""}`,
		),
		"reboot mismatch": []byte(
			`KPANEL_BBRV3_RESULT {"action":"install","changed":false,"rebootRequired":false}` + "\n" +
				`KPANEL_BBRV3_STATUS {"supported":true,"installed":true,"rebootRequired":true,"reason":""}`,
		),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := verifyBBRv3ActionOutput(output, "install"); err == nil {
				t.Fatalf("invalid action output accepted: %s", output)
			}
		})
	}
}

func TestBBRv3StatusUsesFixedKejilionProtocol(t *testing.T) {
	runner := &fakeRunner{
		run: func(_ context.Context, name string, arguments ...string) ([]byte, error) {
			if name != "env" {
				t.Fatalf("BBRv3 status command = %s %#v", name, arguments)
			}
			return []byte(
				"KPANEL_BBRV3_PROTOCOL 1\n" +
					`KPANEL_BBRV3_STATUS {"supported":true,"installed":false,"active":false,"architecture":"x86_64","os":"ubuntu","codename":"noble","runningKernel":"6.8.0-generic","installedKernel":"","congestionControl":"cubic","defaultQDisc":"fq_codel","rebootRequired":false,"reason":""}` +
					"\n",
			), nil
		},
	}
	manager, _, _, _ := testManager(t, runner)
	manager.bbrv3Script = func() (string, error) {
		return "/home/docker/kpanel/bin/kejilion.sh", nil
	}
	status := manager.BBRv3Status(context.Background())
	if !status.Available || !status.Supported || status.Installed {
		t.Fatalf("BBRv3 status = %#v", status)
	}
	command := strings.Join(runner.commands, "\n")
	for _, required := range []string{
		"env KJ_BBRV3_NONINTERACTIVE=1",
		"bash /home/docker/kpanel/bin/kejilion.sh bbrv3 status",
	} {
		if !strings.Contains(command, required) {
			t.Fatalf("BBRv3 command missing %q: %s", required, command)
		}
	}
	for _, forbidden := range []string{"sh -c", "bash -c", "curl"} {
		if strings.Contains(command, forbidden) {
			t.Fatalf("BBRv3 status bypassed the fixed script protocol: %s", command)
		}
	}
}

func TestBBRv3StatusReportsUnavailableProtocol(t *testing.T) {
	manager, _, _, _ := testManager(t, &fakeRunner{})
	manager.bbrv3Script = func() (string, error) {
		return "", errors.New("old script")
	}
	status := manager.BBRv3Status(context.Background())
	if status.Available || status.Reason != "protocol_unavailable" {
		t.Fatalf("unavailable BBRv3 status = %#v", status)
	}
}

func TestBBRv3RunsAsPersistentFixedMaintenanceTask(t *testing.T) {
	runner := &fakeRunner{
		run: func(_ context.Context, name string, arguments ...string) ([]byte, error) {
			if name == "env" {
				action := arguments[len(arguments)-1]
				return []byte(
					`KPANEL_BBRV3_RESULT {"action":"` + action + `","changed":true,"rebootRequired":true}` + "\n" +
						`KPANEL_BBRV3_STATUS {"supported":true,"installed":true,"active":false,"architecture":"x86_64","os":"debian","codename":"bookworm","runningKernel":"6.1.0-amd64","installedKernel":"6.12.0-x64v3-xanmod1","congestionControl":"bbr","defaultQDisc":"fq","rebootRequired":true,"reason":""}` + "\n",
				), nil
			}
			return nil, nil
		},
	}
	manager, _, _, stateDir := testManager(t, runner)
	manager.executable = filepath.Join(stateDir, "kejilion-agent")
	manager.bbrv3Script = func() (string, error) {
		return "/home/docker/kpanel/bin/kejilion.sh", nil
	}

	changed, message, err := manager.startMaintenance(
		context.Background(),
		"bbrv3",
		"install",
	)
	if err != nil || !changed || message == "" {
		t.Fatalf("start BBRv3: changed=%v message=%q err=%v", changed, message, err)
	}
	if err := manager.RunMaintenance(context.Background(), "bbrv3-install"); err != nil {
		t.Fatal(err)
	}
	status := manager.MaintenanceStatus()
	if status.Action != "bbrv3" || status.Policy != "install" ||
		status.State != "succeeded" || status.Progress != 100 || !status.RebootRequired {
		t.Fatalf("BBRv3 maintenance state = %#v", status)
	}
	command := strings.Join(runner.commands, "\n")
	for _, required := range []string{
		"--unit=kejilion-panel-maintenance-",
		manager.executable + " maintenance-run --state-dir " + stateDir + " bbrv3-install",
		"env KJ_BBRV3_NONINTERACTIVE=1 LC_ALL=C.UTF-8 LANG=C.UTF-8 bash " +
			"/home/docker/kpanel/bin/kejilion.sh bbrv3 install",
	} {
		if !strings.Contains(command, required) {
			t.Fatalf("BBRv3 task missing %q:\n%s", required, command)
		}
	}
	for _, forbidden := range []string{"sh -c", "bash -c"} {
		if strings.Contains(command, forbidden) {
			t.Fatalf("BBRv3 task used a shell fragment %q:\n%s", forbidden, command)
		}
	}
}
