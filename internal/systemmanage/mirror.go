package systemmanage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const publicCountryEndpoint = "https://ipinfo.io/country"

type mirrorTarget struct {
	preset   string
	host     string
	label    string
	official bool
}

type mirrorSourceChange struct {
	path    string
	old     []byte
	new     []byte
	mode    os.FileMode
	existed bool
}

var aptURLPattern = regexp.MustCompile(`https?://[A-Za-z0-9.-]+(?::[0-9]+)?/[^ \t\r\n"'<>]+`)

// linuxMirrorsHosts is an audited allowlist from the mirror choices exposed by
// LinuxMirrors. It lets KPanel safely recognize and replace distribution URLs
// previously written by kejilion.sh, while leaving third-party repositories
// such as Docker and NodeSource untouched.
var linuxMirrorsHosts = stringSet(
	"deb.debian.org", "security.debian.org", "ftp.debian.org",
	"archive.ubuntu.com", "security.ubuntu.com", "ports.ubuntu.com",
	"mirrors.aliyun.com", "mirrors.cloud.tencent.com", "mirrors.tencent.com",
	"mirrors.huaweicloud.com", "repo.huaweicloud.com", "mirrors.cmecloud.cn",
	"mirrors.ctyun.cn", "mirrors.163.com", "mirrors.volces.com",
	"mirrors.tuna.tsinghua.edu.cn", "mirrors.pku.edu.cn", "mirrors.zju.edu.cn",
	"mirrors.nju.edu.cn", "mirror.lzu.edu.cn", "mirror.sjtu.edu.cn",
	"mirrors.hust.edu.cn", "mirrors.ustc.edu.cn", "mirror.iscas.ac.cn",
	"mirrors.cstcloud.cn", "mirror.bjtu.edu.cn", "mirrors.bfsu.edu.cn",
	"mirrors.bupt.edu.cn", "mirrors.cqu.edu.cn", "mirrors.cqupt.edu.cn",
	"mirrors.neusoft.edu.cn", "mirrors.uestc.cn", "mirrors.scau.edu.cn",
	"mirrors.jlu.edu.cn", "mirrors.jcut.edu.cn", "mirrors.jxust.edu.cn",
	"mirrors.njtech.edu.cn", "mirrors.njupt.edu.cn", "mirrors.sustech.edu.cn",
	"mirror.nyist.edu.cn", "mirrors.qlu.edu.cn", "mirrors.sdu.edu.cn",
	"mirrors.shanghaitech.edu.cn", "mirrors.sjtug.sjtu.edu.cn",
	"mirrors.wsyu.edu.cn", "mirrors.xjtu.edu.cn", "mirrors.nwafu.edu.cn",
	"mirrors.xtom.hk", "mirror.01link.hk", "download.nus.edu.sg",
	"mirror.sg.gs", "mirrors.xtom.sg", "free.nchc.org.tw",
	"mirror.ossplanet.net", "linux.cs.nycu.edu.tw", "ftp.tku.edu.tw",
	"mirror.twds.com.tw", "mirror.anigil.com", "ftp.udx.icscoe.jp",
	"ftp.jaist.ac.jp", "linux2.yz.yamagata-u.ac.jp", "mirrors.xtom.jp",
	"mirrors.gbnetwork.com", "mirror.kku.ac.th", "mirror.vorboss.net",
	"mirror.quickhost.uk", "mirror.dogado.de", "mirrors.xtom.de",
	"ftp.halifax.rwth-aachen.de", "ftp.agdsn.de", "mirror.in2p3.fr",
	"mirrors.ircam.fr", "eclats.crans.org", "ftp.crihan.fr",
	"mirrors.xtom.nl", "mirror.datapacket.com", "eu.edge.kernel.org",
	"mirrors.xtom.ee", "mirror.netsite.dk", "mirrors.dotsrc.org",
	"mirror.accum.se", "ftp.lysator.liu.se", "mirror.yandex.ru",
	"mirror.linux-ia64.org", "mirror.truenetwork.ru", "ftp.belnet.be",
	"ftp.cc.uoc.gr", "ftp.fi.muni.cz", "ftp.sh.cvut.cz",
	"mirror.karneval.cz", "mirrors.nic.cz", "mirror.ethz.ch",
	"mirrors.kernel.org", "mirrors.mit.edu", "mirror.math.princeton.edu",
	"ftp-chi.osuosl.org", "mirror.fcix.net", "mirrors.xtom.com",
	"mirror.steadfast.net", "mirror.it.ubc.ca", "mirror.xenyth.net",
	"mirrors.switch.ca", "mirror.pop-sc.rnp.br", "mirror.uepg.br",
	"mirror.ufscar.br", "mirrors.eze.sysarmy.com", "gsl-syd.mm.fcix.net",
	"mirror.aarnet.edu.au", "mirror.datamossa.io", "mirror.amaze.com.au",
	"mirrors.xtom.au", "mirror.overthewire.com.au", "mirror.fsmg.org.nz",
	"mirror.liquidtelecom.com", "mirror.dimensiondata.com",
)

func resolvePublicCountry(ctx context.Context) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, publicCountryEndpoint, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "text/plain")
	request.Header.Set("User-Agent", "KPanel mirror selector")
	client := &http.Client{Timeout: 4 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return "", fmt.Errorf("country lookup returned HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 16))
	if err != nil {
		return "", err
	}
	country := strings.ToUpper(strings.TrimSpace(string(data)))
	if len(country) != 2 || country[0] < 'A' || country[0] > 'Z' ||
		country[1] < 'A' || country[1] > 'Z' {
		return "", errors.New("country lookup returned an invalid country code")
	}
	return country, nil
}

func (m *Manager) selectMirrorTarget(ctx context.Context, preset string) (mirrorTarget, error) {
	switch preset {
	case "cn-default":
		return mirrorTarget{
			preset: preset, host: "mirrors.aliyun.com",
			label: "中国大陆默认线路（阿里云）",
		}, nil
	case "cn-edu":
		return mirrorTarget{
			preset: preset, host: "mirrors.pku.edu.cn",
			label: "中国大陆教育网线路（北京大学）",
		}, nil
	case "abroad":
		return mirrorTarget{
			preset: preset, host: "mirrors.xtom.hk",
			label: "海外线路（xTom 香港）",
		}, nil
	case "smart":
		country, err := m.country(ctx)
		if err == nil && country == "CN" {
			return mirrorTarget{
				preset: preset, host: "mirrors.huaweicloud.com",
				label: "智能线路（中国大陆 · 华为云）",
			}, nil
		}
		target := mirrorTarget{
			preset: preset, official: true,
			label: "智能线路（海外 · 发行版官方源）",
		}
		if err != nil {
			target.label = "智能线路（地区识别失败，安全回退发行版官方源）"
		}
		return target, nil
	// Keep the v0.11 API values as undocumented compatibility aliases so an
	// older Panel cannot break a newly upgraded Agent.
	case "official":
		return mirrorTarget{
			preset: preset, official: true,
			label: "发行版官方源",
		}, nil
	case "aliyun":
		return mirrorTarget{
			preset: preset, host: "mirrors.aliyun.com",
			label: "中国大陆默认线路（阿里云）",
		}, nil
	default:
		return mirrorTarget{}, fmt.Errorf(
			"%w: mirrorPreset must be cn-default, cn-edu, abroad, or smart",
			ErrInvalidInput,
		)
	}
}

func (m *Manager) setMirror(ctx context.Context, preset string) (bool, string, string, error) {
	osID := strings.ToLower(osReleaseValue(filepath.Join(m.etcRoot, "os-release"), "ID"))
	if osID != "debian" && osID != "ubuntu" {
		return false, "", "", fmt.Errorf(
			"%w: safe mirror switching currently supports Debian and Ubuntu",
			ErrUnsupported,
		)
	}
	target, err := m.selectMirrorTarget(ctx, preset)
	if err != nil {
		return false, "", "", err
	}
	files := m.aptSourceFiles()
	if len(files) == 0 {
		return false, "", "", fmt.Errorf("%w: no APT source files were found", ErrUnsupported)
	}

	recognized := 0
	var changes []mirrorSourceChange
	for _, path := range files {
		old, existed, mode, err := snapshotFile(path)
		if err != nil {
			return false, "", "", err
		}
		rewritten, count := rewriteAPTSourceTarget(old, osID, target)
		recognized += count
		if !bytes.Equal(old, rewritten) {
			changes = append(changes, mirrorSourceChange{
				path: path, old: old, new: rewritten, mode: mode, existed: existed,
			})
		}
	}
	if recognized == 0 {
		return false, "", "", fmt.Errorf(
			"%w: no recognized Debian/Ubuntu distribution repositories were found; custom and third-party sources were left unchanged",
			ErrConflict,
		)
	}
	if len(changes) == 0 {
		return false, "", "软件源已经使用" + target.label + "，无需变更", nil
	}

	paths := make([]string, 0, len(changes))
	for _, change := range changes {
		paths = append(paths, change.path)
	}
	backup, err := m.createBackup("mirror-"+target.preset, paths...)
	if err != nil {
		return false, "", "", err
	}
	restore := func() error {
		var restoreErrors []error
		for _, change := range changes {
			if err := restoreFile(change.path, change.old, change.existed, change.mode); err != nil {
				restoreErrors = append(restoreErrors, err)
			}
		}
		return errors.Join(restoreErrors...)
	}
	for _, change := range changes {
		if err := writeAtomic(change.path, change.new, fileModeOr(change.mode, 0o644)); err != nil {
			_ = restore()
			return false, backup, "", fmt.Errorf("%w: write APT source: %v", ErrRolledBack, err)
		}
	}

	aptState := filepath.Join(m.stateDir, "apt-validation")
	aptLists := filepath.Join(aptState, "lists")
	aptCache := filepath.Join(aptState, "cache")
	for _, path := range []string{
		filepath.Join(aptLists, "partial"),
		filepath.Join(aptCache, "archives", "partial"),
	} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			_ = restore()
			return false, backup, "", fmt.Errorf(
				"%w: prepare isolated APT validation state: %v",
				ErrRolledBack,
				err,
			)
		}
	}
	if _, err := m.runner.Run(
		ctx, "apt-get", "-o", "Acquire::Retries=1",
		"-o", "Acquire::http::Timeout=12", "-o", "Acquire::https::Timeout=12",
		"-o", "Dir::State::Lists="+aptLists,
		"-o", "Dir::Cache="+aptCache,
		"-o", "APT::Get::List-Cleanup=1",
		"update",
	); err != nil {
		if rollbackErr := restore(); rollbackErr != nil {
			return false, backup, "", fmt.Errorf(
				"%w: APT validation failed and rollback failed: %v",
				ErrNeedsAttention,
				rollbackErr,
			)
		}
		return false, backup, "", fmt.Errorf("%w: APT validation failed: %v", ErrRolledBack, err)
	}
	return true, backup,
		"已切换为" + target.label + "并通过隔离 apt-get update 验证；未升级软件、未清缓存，第三方源未修改",
		nil
}

func rewriteAPTSource(data []byte, osID, preset string) []byte {
	target := mirrorTarget{preset: preset}
	switch preset {
	case "official":
		target.official = true
	case "cn-default", "aliyun":
		target.host = "mirrors.aliyun.com"
	case "cn-edu":
		target.host = "mirrors.pku.edu.cn"
	case "abroad":
		target.host = "mirrors.xtom.hk"
	default:
		return append([]byte(nil), data...)
	}
	rewritten, _ := rewriteAPTSourceTarget(data, osID, target)
	return rewritten
}

func rewriteAPTSourceTarget(data []byte, osID string, target mirrorTarget) ([]byte, int) {
	recognized := 0
	rewritten := aptURLPattern.ReplaceAllFunc(data, func(raw []byte) []byte {
		parsed, err := url.Parse(string(raw))
		if err != nil || !linuxMirrorsHosts[strings.ToLower(parsed.Hostname())] {
			return raw
		}
		branch, suffix, ok := aptDistributionPath(parsed.Path, osID)
		if !ok {
			return raw
		}
		recognized++
		host := target.host
		if target.official {
			host = officialAPTHost(osID, branch)
		}
		if host == "" {
			return raw
		}
		return []byte("https://" + host + "/" + branch + suffix)
	})
	return rewritten, recognized
}

func aptDistributionPath(path, osID string) (string, string, bool) {
	segments := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for index, segment := range segments {
		switch osID {
		case "debian":
			if segment != "debian" && segment != "debian-security" {
				continue
			}
		case "ubuntu":
			if segment != "ubuntu" && segment != "ubuntu-ports" {
				continue
			}
		default:
			return "", "", false
		}
		suffix := ""
		if index+1 < len(segments) {
			suffix = "/" + strings.Join(segments[index+1:], "/")
		}
		if strings.HasSuffix(path, "/") && !strings.HasSuffix(suffix, "/") {
			suffix += "/"
		}
		return segment, suffix, true
	}
	return "", "", false
}

func officialAPTHost(osID, branch string) string {
	switch osID {
	case "debian":
		if branch == "debian-security" {
			return "security.debian.org"
		}
		return "deb.debian.org"
	case "ubuntu":
		if branch == "ubuntu-ports" {
			return "ports.ubuntu.com"
		}
		return "archive.ubuntu.com"
	default:
		return ""
	}
}

func stringSet(values ...string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}
