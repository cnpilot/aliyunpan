// Copyright (c) 2020 tickstep.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package command

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/plugins"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/library-go/logger"
	"math/rand"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var (
	panCommandVerbose = logger.New("PANCOMMAND", config.EnvVerbose)
)

// RunTestShellPattern 执行测试通配符
func RunTestShellPattern(driveId string, pattern string) {
	acUser := GetActiveUser()
	files, err := acUser.PanClient().MatchPathByShellPattern(driveId, GetActiveUser().PathJoin(driveId, pattern))
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, f := range *files {
		fmt.Printf("%s\n", f.Path)
	}
	return
}

// matchPathByShellPattern 通配符匹配路径，允许返回多个匹配结果
func matchPathByShellPattern(driveId string, patterns ...string) (files []*aliyunpan.FileEntity, e error) {
	acUser := GetActiveUser()
	for k := range patterns {
		ps, err := acUser.PanClient().MatchPathByShellPattern(driveId, acUser.PathJoin(driveId, patterns[k]))
		if err != nil {
			return nil, err
		}
		files = append(files, *ps...)
	}
	return files, nil
}

// makePathAbsolute 拼接路径，确定返回路径为绝对路径
func makePathAbsolute(driveId string, patterns ...string) (panpaths []string, err error) {
	acUser := GetActiveUser()
	for k := range patterns {
		ps := acUser.PathJoin(driveId, patterns[k])
		panpaths = append(panpaths, ps)
	}
	return panpaths, nil
}

func RandomStr(count int) string {
	//STR_SET := "abcdefjhijklmnopqrstuvwxyzABCDEFJHIJKLMNOPQRSTUVWXYZ1234567890"
	STR_SET := "abcdefjhijklmnopqrstuvwxyz1234567890"
	rand.Seed(time.Now().UnixNano())
	str := strings.Builder{}
	for i := 0; i < count; i++ {
		str.WriteByte(byte(STR_SET[rand.Intn(len(STR_SET))]))
	}
	return str.String()
}

func GetAllPathFolderByPath(pathStr string) []string {
	dirNames := strings.Split(pathStr, "/")
	dirs := []string{}
	p := "/"
	dirs = append(dirs, p)
	for _, s := range dirNames {
		p = path.Join(p, s)
		dirs = append(dirs, p)
	}
	return dirs
}

// EscapeStr 转义字符串
func EscapeStr(s string) string {
	return url.PathEscape(s)
}

// UnescapeStr 反转义字符串
func UnescapeStr(s string) string {
	r, _ := url.PathUnescape(s)
	return r
}

// RefreshTokenInNeed 刷新refresh token
func RefreshTokenInNeed(activeUser *config.PanUser) bool {
	if activeUser == nil {
		return false
	}

	// refresh expired token
	if activeUser.PanClient() != nil {
		if len(activeUser.WebToken.RefreshToken) > 0 {
			cz := time.FixedZone("CST", 8*3600) // 东8区
			expiredTime, _ := time.ParseInLocation("2006-01-02 15:04:05", activeUser.WebToken.ExpireTime, cz)
			now := time.Now()
			if (expiredTime.Unix() - now.Unix()) <= (20 * 60) { // 20min
				// need update refresh token
				logger.Verboseln("access token expired, get new from refresh token")
				if wt, er := aliyunpan.GetAccessTokenFromRefreshToken(activeUser.RefreshToken); er == nil {
					activeUser.RefreshToken = wt.RefreshToken
					activeUser.WebToken = *wt
					activeUser.PanClient().UpdateToken(*wt)
					logger.Verboseln("get new access token success")
					return true
				} else {
					// token refresh error
					// if token has expired, callback plugin api for notify
					if now.Unix() >= expiredTime.Unix() {
						pluginManger := plugins.NewPluginManager(config.GetPluginDir())
						plugin, _ := pluginManger.GetPlugin()
						params := &plugins.UserTokenRefreshFinishParams{
							Result:    "fail",
							Message:   er.Error(),
							OldToken:  activeUser.RefreshToken,
							NewToken:  "",
							UpdatedAt: utils.NowTimeStr(),
						}
						if er1 := plugin.UserTokenRefreshFinishCallback(plugins.GetContext(activeUser), params); er1 != nil {
							logger.Verbosef("UserTokenRefreshFinishCallback error: " + er1.Error())
						}
					}
				}
			}
		}
	}
	return false
}

func isIncludeFile(pattern string, fileName string) bool {
	b, er := filepath.Match(pattern, fileName)
	if er != nil {
		return false
	}
	return b
}

// isMatchWildcardPattern 是否是统配符字符串
func isMatchWildcardPattern(name string) bool {
	return strings.ContainsAny(name, "*") || strings.ContainsAny(name, "?") || strings.ContainsAny(name, "[")
}
