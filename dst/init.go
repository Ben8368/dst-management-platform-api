package dst

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/internal/runner"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"sync"
)

type Game struct {
	room    *models.Room
	worlds  *[]models.World
	setting *models.RoomSetting
	lang    string
	roomSaveData
	worldSaveData []worldSaveData
	playerSaveData
	modSaveData
	// room全局文件锁
	roomMutex sync.Mutex
	// world全局文件锁
	worldMutex sync.Mutex
	// player全局文件锁
	playerMutex sync.Mutex
	// acf文件锁
	acfMutex sync.Mutex
	// mod 文件、map锁
	modMutex sync.Mutex
}

func NewGameController(room *models.Room, worlds *[]models.World, setting *models.RoomSetting, lang string) *Game {
	game := &Game{
		room:    room,
		worlds:  worlds,
		setting: setting,
		lang:    lang,
	}

	game.initInfo()

	return game
}

func (g *Game) initInfo() {
	// room
	g.clusterName = fmt.Sprintf("Cluster_%d", g.room.ID)
	g.clusterPath = fmt.Sprintf("%s/%s", utils.ClusterPath, g.clusterName)
	g.clusterIniPath = fmt.Sprintf("%s/cluster.ini", g.clusterPath)
	g.clusterTokenTxtPath = fmt.Sprintf("%s/cluster_token.txt", g.clusterPath)

	// worlds
	config := runner.DefaultConfig()
	for _, world := range *g.worlds {
		customGameStartupCmd := world.CustomStartupCmd
		// 自定义启动命令会被拼接进 bash -c 执行，不合法时一律忽略，使用默认启动命令
		if customGameStartupCmd != "" && !utils.IsValidGameStartupCmd(customGameStartupCmd) {
			logger.Logger.Warnf("自定义启动命令不合法，已忽略, world: %s, cmd: %s", world.WorldName, customGameStartupCmd)
			customGameStartupCmd = ""
		}
		worldPath := fmt.Sprintf("%s/%s", g.clusterPath, world.WorldName)
		serverIniPath := fmt.Sprintf("%s/server.ini", worldPath)
		savePath := fmt.Sprintf("%s/save", worldPath)
		sessionPath := fmt.Sprintf("%s/session", savePath)
		levelDataOverridePath := fmt.Sprintf("%s/leveldataoverride.lua", worldPath)
		modOverridesPath := fmt.Sprintf("%s/modoverrides.lua", worldPath)
		screenName := fmt.Sprintf("DMP_%s_%s", g.clusterName, world.WorldName)

		startCmd, startCmdErr := g.generateGameStartCmd(screenName, customGameStartupCmd, world.WorldName, config)
		if startCmdErr != nil {
			logger.Logger.Errorf("生成世界启动命令失败, world: %s, err: %v", world.WorldName, startCmdErr)
		}

		g.worldSaveData = append(g.worldSaveData, worldSaveData{
			worldPath:             worldPath,
			serverIniPath:         serverIniPath,
			savePath:              savePath,
			sessionPath:           sessionPath,
			levelDataOverridePath: levelDataOverridePath,
			modOverridesPath:      modOverridesPath,
			startCmd:              startCmd,
			startCmdErr:           startCmdErr,
			screenName:            screenName,
			World:                 world,
		})
	}

	// players
	g.adminlistPath = fmt.Sprintf("%s/adminlist.txt", g.clusterPath)
	g.whitelistPath = fmt.Sprintf("%s/whitelist.txt", g.clusterPath)
	g.blocklistPath = fmt.Sprintf("%s/blocklist.txt", g.clusterPath)
	g.adminlist = getPlayerList(g.adminlistPath)
	g.whitelist = getPlayerList(g.whitelistPath)
	g.blocklist = getPlayerList(g.blocklistPath)

	// mods
	if cache.OsType == utils.Darwin {
		g.ugcPath = fmt.Sprintf("%s/%s/%s", cache.CurrentDir, utils.DarwinGameUgcPath, g.clusterName)
		g.notUgcPath = utils.DarwinGameNotUgcPath
	} else {
		g.ugcPath = fmt.Sprintf("%s/%s/%s", cache.CurrentDir, utils.GameUgcPath, g.clusterName)
		g.notUgcPath = utils.GameNotUgcPath
	}

}

func (g *Game) generateGameStartCmd(screenName, customGameStartupCmd, worldName string, config runner.Config) (string, error) {
	// screenName来自数据库/存档，统一使用ShellQuote，避免被拼接进bash后解释执行
	quotedScreenName := utils.ShellQuote(screenName)

	// Build DST command spec with compatibility layer
	var binDir string
	var binary string
	var targetArch runner.BinaryArch

	if config.HostOS == utils.Darwin {
		// Darwin: native execution
		binDir = utils.DarwinDstBinDir
		binary = "./dontstarve_dedicated_server_nullrenderer"
		targetArch = runner.Native
	} else {
		// Linux: determine binary and architecture based on StartType
		switch g.setting.StartType {
		case "32-bit":
			binDir = "dst/bin/"
			binary = "./dontstarve_dedicated_server_nullrenderer"
			targetArch = runner.X8632
		case "64-bit":
			binDir = "dst/bin64/"
			binary = "./dontstarve_dedicated_server_nullrenderer_x64"
			targetArch = runner.X8664
		case "luajit":
			binDir = "dst/bin64/"
			binary = "./dontstarve_dedicated_server_nullrenderer_x64_luajit"
			targetArch = runner.X8664
		default:
			return "", fmt.Errorf("unsupported StartType: %s", g.setting.StartType)
		}
	}

	// Build DST arguments
	args := []string{
		"-console",
		"-cluster",
		g.clusterName,
		"-shard",
		worldName,
	}

	// Build compatibility command
	spec, err := runner.BuildCompatCommand(binary, args, targetArch, config)
	if err != nil {
		return "", fmt.Errorf("failed to build compatibility command: %w", err)
	}

	// Render to shell command
	dstCmd := runner.ShellRender(spec)

	// Build full screen command
	// customGameStartupCmd is a trusted/validated shell fragment retained for backward compatibility
	var startCmd string
	if config.HostOS == utils.Darwin {
		startCmd = fmt.Sprintf("export DYLD_LIBRARY_PATH=$DYLD_LIBRARY_PATH:%s/steamcmd && cd %s && screen -d -h 200 -m -S %s %s %s",
			cache.CurrentDir, binDir, quotedScreenName, customGameStartupCmd, dstCmd)
	} else {
		startCmd = fmt.Sprintf("cd %s && screen -d -h 200 -m -S %s %s %s",
			binDir, quotedScreenName, customGameStartupCmd, dstCmd)
	}

	return startCmd, nil
}
