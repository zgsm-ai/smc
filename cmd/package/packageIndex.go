package pkg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zgsm-ai/smc/internal/utils"
)

/**
 *	platform.json
 */
type PlatformNode struct {
	BaseDir  string
	Os       string
	Arch     string
	Newest   *utils.PackageVersion
	Versions []*utils.PackageVersion
}

/**
 *	platforms.json
 */
type PackageNode struct {
	BaseDir     string
	PackageName string
	Platforms   map[string]*PlatformNode
}

/**
 *	packages.json
 */
type PackagesNode struct {
	BaseDir  string
	Packages map[string]*PackageNode
}

/**
 *	Package tree
 */
var allPackages PackagesNode = PackagesNode{
	Packages: make(map[string]*PackageNode),
}

/**
 *	Add a package with filename fpath (with package.json) to the package list
 */
func addPackage(fpath string) {
	//<packageName>/windows/amd64/1.1.1125/package.json
	pkgVer := &utils.PackageVersion{}
	bytes, err := os.ReadFile(fpath)
	if err != nil {
		fmt.Printf("read error, ignore %s\n", fpath)
		return
	}
	if err = json.Unmarshal(bytes, &pkgVer); err != nil {
		fmt.Printf("unmarshal error, ignore %s\n", fpath)
		return
	}
	fmt.Printf("found package.json: %s\n", fpath)

	if len(allPackages.Packages) == 0 {
		allPackages.BaseDir = getPackagesDir(fpath)
	}
	pkg, ok := allPackages.Packages[pkgVer.PackageName]
	if !ok {
		pkg = &PackageNode{
			PackageName: pkgVer.PackageName,
			BaseDir:     getPlatformsDir(fpath),
			Platforms:   make(map[string]*PlatformNode),
		}
		allPackages.Packages[pkgVer.PackageName] = pkg
	}
	keyStr := fmt.Sprintf("%s-%s", pkgVer.Os, pkgVer.Arch)
	platform, ok := pkg.Platforms[keyStr]
	if !ok {
		platform = &PlatformNode{
			BaseDir: getPlatformDir(fpath),
			Os:      pkgVer.Os,
			Arch:    pkgVer.Arch,
		}
		pkg.Platforms[keyStr] = platform
	}
	platform.Versions = append(platform.Versions, pkgVer)
}

/**
 *	Get version info of the newest package
 */
func getNewest() {
	for _, pkg := range allPackages.Packages {
		for _, plat := range pkg.Platforms {
			var newest *utils.PackageVersion
			for _, v := range plat.Versions {
				if newest == nil || utils.CompareVersion(v.VersionId, newest.VersionId) > 0 {
					newest = v
				}
			}
			plat.Newest = newest
		}
	}
}

func savePlatform(pkname string, node *PlatformNode) error {
	plat := getPlatformInfo(pkname, node)

	data, err := json.MarshalIndent(plat, "", "  ")
	if err != nil {
		fmt.Println(err)
		return err
	}
	fpath := filepath.Join(node.BaseDir, "platform.json")

	fmt.Printf("create %s, versions: %d, newest: %s\n", fpath,
		len(plat.Versions), utils.PrintVersion(plat.Newest.VersionId))
	if err = os.WriteFile(fpath, data, 0666); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func savePlatforms(plats *PackageNode) error {
	fpath := filepath.Join(plats.BaseDir, "platforms.json")
	if !isSubdirectory(plats.BaseDir, optBuildDir) {
		fmt.Printf("ignore %s\n", fpath)
		return nil
	}
	var platforms utils.PackageDirectory
	platforms.PackageName = plats.PackageName
	for _, v := range plats.Platforms {
		platforms.Platforms = append(platforms.Platforms, utils.PlatformId{
			Os:   v.Os,
			Arch: v.Arch,
		})
	}
	fmt.Printf("create %s, platforms: %d\n", fpath, len(platforms.Platforms))
	data, err := json.MarshalIndent(platforms, "", "  ")
	if err != nil {
		fmt.Println(err)
		return err
	}
	if err := os.WriteFile(fpath, data, 0666); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func getVersionAddr(pkgVer *utils.PackageVersion) utils.VersionAddr {
	ver := &utils.VersionAddr{}
	ver.VersionId = pkgVer.VersionId
	verStr := utils.PrintVersion(ver.VersionId)
	ver.AppUrl = fmt.Sprintf("/%s/%s/%s/%s/%s",
		pkgVer.PackageName, pkgVer.Os, pkgVer.Arch, verStr, pkgVer.FileName)
	ver.InfoUrl = fmt.Sprintf("/%s/%s/%s/%s/package.json",
		pkgVer.PackageName, pkgVer.Os, pkgVer.Arch, verStr)
	return *ver
}

func getVersionNode(pkgVer *utils.PackageVersion) utils.VersionOverview {
	node := utils.VersionOverview{}

	node.VersionId = pkgVer.VersionId
	node.PackageType = pkgVer.PackageType
	node.Size = pkgVer.Size
	node.FileName = pkgVer.FileName
	node.Description = pkgVer.Description
	node.Build = pkgVer.Build
	return node
}

func getPlatformInfo(pkname string, node *PlatformNode) utils.PlatformInfo {
	var plat utils.PlatformInfo
	plat.Arch = node.Arch
	plat.Os = node.Os
	plat.PackageName = pkname
	plat.Newest = getVersionAddr(node.Newest)
	for _, v := range node.Versions {
		plat.Versions = append(plat.Versions, getVersionAddr(v))
	}
	return plat
}

func getPlatformTree(node *PlatformNode) utils.PlatformOverview {
	var tree utils.PlatformOverview
	tree.Arch = node.Arch
	tree.Os = node.Os
	for _, v := range node.Versions {
		tree.Versions = append(tree.Versions, getVersionNode(v))
	}
	return tree
}

func getPackageTree(p *PackageNode) utils.PackageOverview {
	var tree utils.PackageOverview
	tree.PackageName = p.PackageName
	for _, pl := range p.Platforms {
		plat := getPlatformTree(pl)
		tree.Platforms = append(tree.Platforms, plat)
	}
	return tree
}

func savePackageList() error {
	if allPackages.BaseDir == "" {
		fmt.Printf("ignore packages.json\n")
		return nil
	}
	if !isSubdirectory(allPackages.BaseDir, optBuildDir) {
		fmt.Printf("ignore packages.json\n")
		return nil
	}
	var list utils.PackageList
	for _, p := range allPackages.Packages {
		list.Packages = append(list.Packages, p.PackageName)
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		fmt.Println(err)
		return err
	}
	fname := filepath.Join(allPackages.BaseDir, "packages.json")
	fmt.Printf("create %s, packages: %d\n", fname, len(allPackages.Packages))
	if err := os.WriteFile(fname, data, 0666); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func savePackagesOverview() error {
	if allPackages.BaseDir == "" {
		fmt.Printf("ignore packages-overview.json\n")
		return nil
	}
	if !isSubdirectory(allPackages.BaseDir, optBuildDir) {
		fmt.Printf("ignore packages-overview.json\n")
		return nil
	}
	var overview utils.PackagesOverview
	overview.Packages = make(map[string]utils.PackageOverview)
	for _, p := range allPackages.Packages {
		overview.Packages[p.PackageName] = getPackageTree(p)
	}
	data, err := json.MarshalIndent(overview, "", "  ")
	if err != nil {
		fmt.Println(err)
		return err
	}
	fname := filepath.Join(allPackages.BaseDir, "packages-overview.json")
	fmt.Printf("create %s, packages: %d\n", fname, len(allPackages.Packages))
	if err := os.WriteFile(fname, data, 0666); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

/**
 *	Save platforms.json, platform.json files to the directory specified by --build
 */
func saveAllPackages() {
	for _, pkg := range allPackages.Packages {
		for _, plat := range pkg.Platforms {
			if err := savePlatform(pkg.PackageName, plat); err != nil {
				fmt.Printf("error: save platform.json failed: %v\n", err)
			}
		}
		if err := savePlatforms(pkg); err != nil {
			fmt.Printf("error: save platforms.json failed: %v\n", err)
		}
	}
	if optPackages {
		if err := savePackageList(); err != nil {
			fmt.Printf("error: save packages.json failed: %v\n", err)
		}
	}
	if optOverview {
		if err := savePackagesOverview(); err != nil {
			fmt.Printf("error: save packages-overview.json failed: %v\n", err)
		}
	}
}

/**
 *	检查 dir 是否是 baseDir 或其子目录
 */
func isSubdirectory(dir, baseDir string) bool {
	if dir == baseDir {
		return true
	}

	rel, err := filepath.Rel(baseDir, dir)
	if err != nil {
		return false
	}

	// 如果相对路径不以 ".." 开头，说明 dir 是 baseDir 或其子目录
	return !filepath.IsAbs(rel) && !startsWithDotDot(rel)
}

/**
 *	检查路径是否以 ".." 开头
 */
func startsWithDotDot(path string) bool {
	return len(path) >= 2 && path[0:2] == ".."
}

/**
 *	Build {package}/{os}/{arch}/packages.json for each platform
 */
func makePackages() error {
	// Traverse all package.json files to build index files
	optBuildDir := filepath.Join(optBuildDir)
	filepath.Walk(optBuildDir, func(fpath string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		if filepath.Base(fpath) != "package.json" {
			return nil
		}
		addPackage(fpath)
		return nil
	})
	// Get the newest version
	getNewest()
	saveAllPackages()
	return nil
}

// parsePackagePath 解析 package.json 所在的路径，返回包含所有各级目录名的数组
func parsePackagePath(fpath string) []string {
	dir := filepath.Dir(fpath)
	var dirs []string
	for dir != "" {
		dirs = append([]string{filepath.Base(dir)}, dirs...)
		if dir == "." || dir == "/" {
			break
		}
		dir = filepath.Dir(dir)
	}
	return dirs
}

func getPlatformDir(fpath string) string {
	// <package>/<os>/<arch>/<ver>/package.json
	names := parsePackagePath(fpath)
	if len(names) > 1 {
		return filepath.Join(names[0 : len(names)-1]...)
	}
	return ""
}

func getPlatformsDir(fpath string) string {
	// <package>/<os>/<arch>/<ver>/package.json
	names := parsePackagePath(fpath)
	if len(names) > 3 {
		return filepath.Join(names[0 : len(names)-3]...)
	}
	return ""
}

func getPackagesDir(fpath string) string {
	// <package>/<os>/<arch>/<ver>/package.json
	names := parsePackagePath(fpath)
	if len(names) > 4 {
		return filepath.Join(names[0 : len(names)-4]...)
	}
	return ""
}

var indexCmd = &cobra.Command{
	Use:   "index {build-dir | -b build-dir}",
	Short: "Generate index files (packages.json/platforms.json/platform.json)",
	Long:  `Scan directorys and generate index files`,

	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// 如果通过args指定了build目录，则覆盖optBuildDir
		if len(args) > 0 {
			optBuildDir = args[0]
		}
		if err := makePackages(); err != nil {
			fmt.Println(err)
		}
	},
}

var optBuildDir string
var optPackages bool
var optOverview bool

func init() {
	packageCmd.AddCommand(indexCmd)

	indexCmd.Example = `  # Scan ./build directory and generate index files based on signed packages
  smc package index -b ./build
  # Or specify build directory as argument
  smc package index ./build`
	indexCmd.Flags().SortFlags = false
	indexCmd.Flags().StringVarP(&optBuildDir, "build", "b", ".", "Build directory: location of package files")
	indexCmd.Flags().BoolVar(&optPackages, "packages", false, "Generate packages.json file")
	indexCmd.Flags().BoolVar(&optOverview, "overview", false, "Generate packages-overview.json file")
}
