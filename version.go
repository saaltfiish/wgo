//
// version.go
// Copyright (C) 2018 Odin <Odin@Odin-Pro.local>
//
// Distributed under terms of the MIT license.
//

package wgo

// The main version number that is being run at the moment.
const WGO_VERSION = "0.9.5"

// A pre-release marker for the version. If this is "" (empty string)
// then it means that it is a final release. Otherwise, this is a pre-release
// such as "dev" (in development)
var VersionPrerelease = ""

// The git commit that was compiled. This will be filled in by the compiler.
var (
	VERSION string

	AppVersion string
	GitCommit  string
	BuildTime  string
	AppLevel   string = "dev" // 产品环境 [dev, testing, production]
)

func init() {
	VERSION = Version()
	Debug("level: %s, version: %s, built at: %s", AppLevel, VERSION, BuildTime)
}

// 获取版本号
func Version() (ver string) {
	// 默认wgo的version
	ver = WGO_VERSION
	if AppVersion != "" {
		ver = AppVersion
	}
	if GitCommit != "" && len(GitCommit) > 8 {
		ver = ver + "-" + GitCommit[:8]
	}
	// todo, support git tags
	return
}

// 获取环境级别
func Level() string {
	return AppLevel
}
