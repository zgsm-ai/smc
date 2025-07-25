package utils

import (
	"bufio"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

/**
 *	版本编号
 */
type VersionNumber struct {
	Major int `json:"major"`
	Minor int `json:"minor"`
	Micro int `json:"micro"`
}

/**
 *	包的描述信息，用于验证包的正确性
 */
type PackageInfo struct {
	PackageName  string        `json:"packageName"`    //包名字
	Os           string        `json:"os"`             //操作系统名:linux/windows
	Arch         string        `json:"arch"`           //硬件架构
	Size         uint64        `json:"size,omitempty"` //包文件大小
	Checksum     string        `json:"checksum"`       //Md5散列值
	Sign         string        `json:"sign"`           //签名，使用私钥签的名，需要用对应公钥验证
	ChecksumAlgo string        `json:"checksumAlgo"`   //固定为“md5”
	VersionId    VersionNumber `json:"versionId"`      //版本号，采用SemVer标准
	Build        string        `json:"build"`          //构建信息：Tag/Branch信息 CommitID BuildTime
	VersionDesc  string        `json:"versionDesc"`    //版本描述，含有更丰富的可读信息
}

/**
 *	一个package版本的地址信息
 */
type VersionAddr struct {
	VersionId  VersionNumber `json:"versionId"`            //版本的地址信息
	AppUrl     string        `json:"appUrl"`               //包地址
	PackageUrl string        `json:"packageUrl,omitempty"` //obsolete field: (过时的)包地址，用来兼容历史版本
	InfoUrl    string        `json:"infoUrl"`              //包描述信息(PackageInfo)文件的地址
}

/**
 *	版本列表，描述一个硬件平台/操作系统对应的包列表
 */
type VersionList struct {
	PackageName string        `json:"packageName"`
	Os          string        `json:"os"`
	Arch        string        `json:"arch"`
	Newest      VersionAddr   `json:"newest"`
	Versions    []VersionAddr `json:"versions"`
}

type UpgradeConfig struct {
	PublicKey   string //用来验证包签名的公钥
	BaseUrl     string //保存安装包的服务器的基地址
	InstallDir  string //软件包的安装路径
	PackageDir  string //保存下载软件包的包描述文件
	PackageName string //包名称
	TargetName  string //目标名称
	Os          string //操作系统名
	Arch        string //硬件平台名
}

const SHENMA_PUBLIC_KEY = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAwClPrRPGCOXcWPFMPIPc
Hn5angPRwuIvwSGle/O7VaZfaTuplMVa2wUPzWv1AfmKpENMm0pf0uhnTyfH3gnR
C46rNeMmBcLg8Jd7wTWXtik0IN7CREOQ6obIiMY4Sbx25EPHPf8SeqvPpFq8uOEM
YqRUQbPaY5+mgkDZMy68hJDUUstapBQovjSlnLXjG2pULWKIJF2g0gGWvS4LGznP
Uvrq2U1QVpsja3EtoLq8jF3UcLJWVZt2pMd5H9m3ULBKFzpu7ix+wb3ebRr6JtUI
bMzLAZ0BM0wxlpDmp1GYVag+Ll3w2o3LXLEB08soABD0wdD03Sp7flkbebgAxd1b
vwIDAQAB
-----END PUBLIC KEY-----`

const SHENMA_BASE_URL = "https://zgsm.sangfor.com/shenma/api/v1"

/**
 *	从AIP获取一个文件
 */
func getBytes(urlStr string, params map[string]string) ([]byte, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return []byte{}, fmt.Errorf("getBytes: %v", err)
	}
	vals := make(url.Values)
	for k, v := range params {
		vals.Set(k, v)
	}
	req.URL.RawQuery = vals.Encode()

	rsp, err := client.Do(req)
	if err != nil {
		return []byte{}, fmt.Errorf("getBytes: %v", err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		rspBody, _ := io.ReadAll(rsp.Body)
		return rspBody, fmt.Errorf("getBytes('%s?%s') code:%d, error:%s",
			urlStr, req.URL.RawQuery, rsp.StatusCode, string(rspBody))
	}
	return io.ReadAll(rsp.Body)
}

/**
 *	创建文件fname依赖的父目录
 */
func mkParentDir(fname string) error {
	dir := filepath.Dir(fname)
	return os.MkdirAll(dir, 0775)
}

/**
 *	从服务器获取一个文件
 */
func getFile(urlStr string, params map[string]string, savePath string) error {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return fmt.Errorf("getFile('%s') failed: %v", urlStr, err)
	}
	vals := make(url.Values)
	for k, v := range params {
		vals.Set(k, v)
	}
	req.URL.RawQuery = vals.Encode()

	rsp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("getFile('%s') failed: %v", urlStr, err)
	}
	defer rsp.Body.Close()
	if rsp.StatusCode != 200 {
		rspBody, _ := io.ReadAll(rsp.Body)
		return fmt.Errorf("getFile('%s', '%s') code: %d, error:%s",
			urlStr, req.URL.RawQuery, rsp.StatusCode, string(rspBody))
	}

	// 创建一个文件用于保存
	if err = mkParentDir(savePath); err != nil {
		return fmt.Errorf("getFile('%s'): mkParentDir('%s') error:%v", urlStr, savePath, err)
	}
	out, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("getFile('%s'): create('%s') error: %v", urlStr, savePath, err)
	}
	defer out.Close()

	// 然后将响应流和文件流对接起来
	_, err = io.Copy(out, rsp.Body)
	if err != nil {
		return fmt.Errorf("getFile('%s'): copy error: %v", urlStr, err)
	}
	return err
}

/**
 *	解析版本字符串，得到版本号
 */
func ParseVersion(verstr string) (VersionNumber, error) {
	vers := strings.Split(verstr, ".")
	id := VersionNumber{}
	if len(vers) != 3 {
		return id, fmt.Errorf("invalid version string")
	}
	var err error
	id.Major, err = strconv.Atoi(vers[0])
	if err != nil {
		return id, fmt.Errorf("invalid version: %v", err)
	}
	id.Minor, err = strconv.Atoi(vers[1])
	if err != nil {
		return id, fmt.Errorf("invalid version: %v", err)
	}
	id.Micro, err = strconv.Atoi(vers[2])
	if err != nil {
		return id, fmt.Errorf("invalid version: %v", err)
	}
	return id, nil
}

/**
 *	打印版本号
 */
func PrintVersion(ver VersionNumber) string {
	return fmt.Sprintf("%d.%d.%d", ver.Major, ver.Minor, ver.Micro)
}

/**
 *	从远程库获取包版本
 */
func GetRemoteVersions(cfg *UpgradeConfig) (VersionList, error) {
	urlStr := fmt.Sprintf("%s/%s/packages-%s-%s.json",
		cfg.BaseUrl, cfg.PackageName, cfg.Os, cfg.Arch)

	bytes, err := getBytes(urlStr, nil)
	if err != nil {
		return VersionList{}, err
	}
	vers := &VersionList{}
	if err = json.Unmarshal(bytes, vers); err != nil {
		return *vers, fmt.Errorf("GetRemoteVersion('%s') unmarshal error: %v", urlStr, err)
	}
	return *vers, nil
}

/**
 *	比较版本
 */
func CompareVersion(local, remote VersionNumber) int {
	if local.Major != remote.Major {
		return local.Major - remote.Major
	}
	if local.Minor != remote.Minor {
		return local.Minor - remote.Minor
	}
	return local.Micro - remote.Micro
}

func correctConfig(cfg *UpgradeConfig) {
	if cfg.PackageName == "" {
		panic("UpgradeConfig.PackageName is emptied")
	}
	if cfg.Arch == "" {
		cfg.Arch = runtime.GOARCH
	}
	if cfg.Os == "" {
		cfg.Os = runtime.GOOS
	}
	fext := ""
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if cfg.InstallDir == "" {
			cfg.InstallDir = filepath.Join(appData, ".costrict\\bin")
		}
		if cfg.PackageDir == "" {
			cfg.PackageDir = filepath.Join(appData, ".costrict\\package")
		}
		fext = ".exe"
	} else if runtime.GOOS == "linux" {
		if cfg.InstallDir == "" {
			cfg.InstallDir = "/usr/local/.costrict/bin"
		}
		if cfg.PackageDir == "" {
			cfg.PackageDir = "/usr/local/.costrict/package"
		}
	}
	if cfg.BaseUrl == "" {
		cfg.BaseUrl = SHENMA_BASE_URL
	}
	if cfg.PublicKey == "" {
		cfg.PublicKey = SHENMA_PUBLIC_KEY
	}
	if cfg.TargetName == "" {
		cfg.TargetName = fmt.Sprintf("%s%s", cfg.PackageName, fext)
	}
}

/**
 *	升级包
 */
func UpgradePackage(cfg *UpgradeConfig, curVer VersionNumber, specVer *VersionNumber) error {
	correctConfig(cfg)
	//	获取AIP平台上的最新版本
	vers, err := GetRemoteVersions(cfg)
	if err != nil {
		return err
	}
	addr := VersionAddr{}
	if specVer != nil { //升级指定版本
		//	检查指定版本specVer在不在版本列表中
		for _, v := range vers.Versions {
			if CompareVersion(v.VersionId, *specVer) == 0 {
				addr = v
				break
			}
		}
		zeroVer := VersionNumber{}
		if CompareVersion(addr.VersionId, zeroVer) == 0 {
			return fmt.Errorf("version %s isn't exist", PrintVersion(*specVer))
		}
	} else { //升级最新版本
		//	比较当前最新版本，看是否有必要升级
		ret := CompareVersion(curVer, vers.Newest.VersionId)
		if ret >= 0 {
			log.Printf("The '%s' version is up to date\n", cfg.PackageName)
			return nil
		}
		addr = vers.Newest
	}
	//	获取AIP平台上升级包的描述信息
	data, err := getBytes(cfg.BaseUrl+addr.InfoUrl, nil)
	if err != nil {
		return err
	}
	pkg := &PackageInfo{}
	if err = json.Unmarshal(data, pkg); err != nil {
		return fmt.Errorf("unmarshal '%s' error: %v", addr.InfoUrl, err)
	}
	//	下载包
	tmpDir, err := os.MkdirTemp("", ".costrict*")
	if err != nil {
		return fmt.Errorf("MkdirTemp error: %v", err)
	}
	tmpFname := filepath.Join(tmpDir, cfg.TargetName)
	if err = getFile(cfg.BaseUrl+addr.AppUrl, nil, tmpFname); err != nil {
		return err
	}
	//	检查下载包的MD5
	_, md5str, err := CalcFileMd5(tmpFname)
	if err != nil {
		return err
	}
	if md5str != pkg.Checksum {
		return fmt.Errorf("checksum error: %s", addr.AppUrl)
	}
	//	检查签名，防止包被篡改
	sig, err := hex.DecodeString(pkg.Sign)
	if err != nil {
		return fmt.Errorf("decode sign error: %v", err)
	}
	if err = VerifySign([]byte(cfg.PublicKey), sig, []byte(md5str)); err != nil {
		return fmt.Errorf("verify sign error: %v", err)
	}
	if err = os.MkdirAll(cfg.InstallDir, 0775); err != nil {
		return fmt.Errorf("MkdirAll('%s') error: %v", cfg.InstallDir, err)
	}
	if err = os.MkdirAll(cfg.PackageDir, 0775); err != nil {
		return fmt.Errorf("MkdirAll('%s') error: %v", cfg.PackageDir, err)
	}
	//	把下载的程序安装到正式目录
	if err = installExecutable(cfg, tmpFname); err != nil {
		return fmt.Errorf("installExecutable('%+v', '%s') error: %v", cfg, tmpFname, err)
	}
	//	把包描述文件保存到包文件目录
	packageFileName := filepath.Join(cfg.PackageDir, fmt.Sprintf("%s.json", cfg.PackageName))
	if err = os.WriteFile(packageFileName, data, 0644); err != nil {
		log.Printf("Write package file(%s) failed: %v", packageFileName, err)
	}
	os.RemoveAll(tmpDir)
	return nil
}

func windowsInstallExecutable(cfg *UpgradeConfig, tmpFname string) error {
	paths := os.Getenv("PATH")
	if !strings.Contains(paths, cfg.InstallDir) {
		newPath := fmt.Sprintf("%s;%s", paths, cfg.InstallDir)
		cmd := exec.Command("setx", "PATH", newPath)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} // 隐藏命令窗口
		if err := cmd.Run(); err != nil {
			return err
		}
		os.Setenv("PATH", newPath)
	}
	targetFileName := filepath.Join(cfg.InstallDir, cfg.TargetName)
	if err := os.Remove(targetFileName); err != nil {
		return err
	}
	if err := os.Rename(tmpFname, targetFileName); err != nil {
		return err
	}
	return os.Chmod(targetFileName, 0755)
}

func linuxInstallExecutable(cfg *UpgradeConfig, tmpFname string) error {
	if err := linuxSetPATH(cfg.InstallDir); err != nil {
		return err
	}
	targetFileName := filepath.Join(cfg.InstallDir, cfg.TargetName)
	if err := os.Rename(tmpFname, targetFileName); err != nil {
		return err
	}
	return os.Chmod(targetFileName, 0755)
}

func linuxSetPATH(installDir string) error {
	currentPath := os.Getenv("PATH")
	// 检查是否已经包含该路径
	currentPathStr := strings.TrimSpace(currentPath)
	if strings.Contains(currentPathStr, installDir) {
		log.Println("The path is already in PATH.")
		return nil
	}
	// 将新路径添加到 PATH
	newPathStr := fmt.Sprintf("%s:%s", currentPathStr, installDir)
	err := os.Setenv("PATH", newPathStr)
	if err != nil {
		fmt.Println("Failed to set PATH for current process:", err)
		return err
	}
	// 获取当前用户的主目录
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Println("Failed to get user home directory:", err)
		return err
	}
	envLine := fmt.Sprintf("export PATH=$PATH:%s", installDir)

	bashrcPath := homeDir + "/.bashrc"
	// 检查是否已经包含该环境变量
	file, err := os.Open(bashrcPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Println("Failed to open ~/.bashrc:", err)
			return err
		}
		// 文件不存在，创建一个空文件
		file, err = os.Create(bashrcPath)
		if err != nil {
			log.Println("Failed to create ~/.bashrc:", err)
			return err
		}
		file.Close()
	} else {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), envLine) {
				file.Close()
				log.Println("Environment variable already exists in ~/.bashrc.")
				return nil
			}
		}
		file.Close()
		if err := scanner.Err(); err != nil {
			log.Println("Failed to read ~/.bashrc:", err)
			return err
		}
	}
	// 将环境变量追加到 ~/.bashrc 文件
	file, err = os.OpenFile(bashrcPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("Failed to open ~/.bashrc for appending:", err)
		return err
	}
	defer file.Close()

	_, err = file.WriteString(envLine + "\n")
	if err != nil {
		log.Println("Failed to write environment variable to ~/.bashrc:", err)
		return err
	}

	log.Println("Environment variable added to ~/.bashrc successfully.")
	return nil
}

/**
 *	把可执行文件安装到应用目录，并保证该程序为可执行状态(比如需要设置PATH和权限位)
 */
func installExecutable(cfg *UpgradeConfig, tmpFname string) error {
	if runtime.GOOS == "windows" {
		return windowsInstallExecutable(cfg, tmpFname)
	} else {
		return linuxInstallExecutable(cfg, tmpFname)
	}
}
