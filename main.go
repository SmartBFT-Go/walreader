/*
Copyright LLC Newity. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"flag"
	"os"

	"github.com/SmartBFT-Go/walreader/reader"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	pathForRead := flag.String("f", "", "path to WAL file")
	pathForSave := flag.String("to", "", "path to the file for saving WAL file")
	flag.Parse()

	config := zap.NewDevelopmentConfig()
	config.DisableCaller = true
	config.Encoding = "console"

	if *pathForSave != "" {
		config.OutputPaths = []string{"stdout", *pathForSave}
	} else {
		config.OutputPaths = []string{"stdout"}
	}

	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	atom := zap.NewAtomicLevel()
	atom.SetLevel(zap.DebugLevel)

	log, err := config.Build()
	if err != nil {
		panic(err)
	}

	logger := log.Sugar()
	defer logger.Sync()

	if *pathForRead == "" {
		logger.Fatal("Please provide path to WAL file of directory with WAL files with -f= flag")
	}

	isD, err := isDir(*pathForRead)
	if err != nil {
		logger.Fatal(err)
	}

	r := reader.NewReader(logger, *pathForRead)
	if !isD {
		if err = r.ReadFile(*pathForRead); err != nil {
			logger.Warnf(err.Error())
		}
	} else {
		if err = r.ReadDir(); err != nil {
			logger.Warnf(err.Error())
		}
	}
}

func isDir(fsPath string) (bool, error) {
	file, err := os.Open(fsPath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return false, err
	}

	return fileInfo.IsDir(), nil
}
