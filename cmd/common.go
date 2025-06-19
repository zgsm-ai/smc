/*
Copyright © 2022 zbc <zbc@sangfor.com.cn>
*/
package cmd

import (
	"fmt"
	"log"

	"github.com/zgsm-ai/smc/internal/env"
	"github.com/zgsm-ai/smc/internal/rc"
	"github.com/zgsm-ai/smc/internal/utils"
)

/**
 * Debug option from command line flags
 */
var optDebug string

/**
 * Log file path from command line flags
 */
var optLogFile string

/**
 * Initialize debug logging configuration
 * @param debug debug level (Off/Err/Dbg)
 * @param logfile path to log file
 * @return error if log initialization fails
 */
func InitDebug(debug, logfile string) error {
	env.InitEnvs()

	if debug == "" {
		debug = env.Debug
	}
	if !env.InNcaseSet(debug, "Off", "Err", "Dbg") {
		log.Printf("debug = %s, expect Off,Err,Dbg\n", debug)
		debug = "Err"
	}
	if logfile == "" {
		logfile = env.Logfile
	}
	if err := env.InitLogger(debug, logfile); err != nil {
		fmt.Println(err)
	}
	return nil
}

/**
 * Global session for communication with AIP platform
 */
var Session *utils.Session

/**
 * Initialize task daemon environment
 * @return error if initialization fails
 */
func InitTaskdEnv() error {
	if Session != nil {
		log.Panicf("Env already init")
	}
	if err := InitDebug(optDebug, optLogFile); err != nil {
		return err
	}
	ss := utils.NewSession(env.TaskdAddr)
	Session = ss
	return nil
}

/**
 * Initialize Redis connection environment
 * @return error if Redis connection fails
 */
func InitRedisEnv() error {
	if err := InitDebug(optDebug, optLogFile); err != nil {
		return err
	}
	if err := rc.InitRedis(env.RedisAddr, env.RedisPwd, env.RedisDb); err != nil {
		return err
	}
	return nil
}

/**
 * Initialize prompt shell environment
 * @return error if initialization fails
 */
func InitPromptEnv() error {
	if err := InitDebug(optDebug, optLogFile); err != nil {
		return err
	}
	ss := utils.NewSession(env.PromptAddr)
	Session = ss
	return nil
}
