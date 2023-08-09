package cmd

import (
	"fmt"
	"github.com/Ionian-Web3-Storage/ionian-client/file"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/ioutil"
	"math/rand"
	"os"
	"time"
)

var (
	genFileArgs struct {
		size      uint64
		file      string
		overwrite bool
	}

	genFileCmd = &cobra.Command{
		Use:   "gen",
		Short: "Generate a temp file for test purpose",
		Run:   generateTempFile,
	}
)

func init() {
	genFileCmd.Flags().Uint64Var(&genFileArgs.size, "size", 0, "File size in bytes")
	genFileCmd.Flags().StringVar(&genFileArgs.file, "file", "", "File name to generate")
	genFileCmd.Flags().BoolVar(&genFileArgs.overwrite, "overwrite", true, "Whether to overwrite existing file")

	rootCmd.AddCommand(genFileCmd)
}

func generateTempFile(*cobra.Command, []string) {
	exists, err := file.Exists(genFileArgs.file)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to check file existence")
	}

	if exists {
		if !genFileArgs.overwrite {
			logrus.Warn("File already exists")
			return
		}

		logrus.Info("Overwrite file")
	}
	if genFileArgs.size == 0 {
		rand.Seed(time.Now().UnixNano())
		// [1M, 10M)
		genFileArgs.size = 1024*1024 + uint64(9.0*1024*1024*rand.Float64())
	}
	if genFileArgs.file == "" {
		fileNameBySize := ""
		if genFileArgs.size < 1024 {
			fileNameBySize = fmt.Sprintf("file%dByte.bin", genFileArgs.size)
		} else if genFileArgs.size < 1024*1024 {
			fileNameBySize = fmt.Sprintf("file%dKB.bin", genFileArgs.size/1024)
		} else {
			fileNameBySize = fmt.Sprintf("file%dMB.bin", genFileArgs.size/1024/1024)
		}
		genFileArgs.file = fileNameBySize
	}

	data := make([]byte, genFileArgs.size)
	if n, err := rand.Read(data); err != nil {
		logrus.WithError(err).Fatal("Failed to generate random data")
	} else if n != len(data) {
		logrus.WithField("n", n).Fatal("Invalid data len")
	}

	if err = ioutil.WriteFile(genFileArgs.file, data, os.ModePerm); err != nil {
		logrus.WithError(err).Fatal("Failed to write file")
	}

	file, err := file.Open(genFileArgs.file)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to open file")
	}

	tree, err := file.MerkleTree()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to generate merkle tree")
	}
	logrus.WithField("file", genFileArgs.file).Info("Write to file")
	logrus.WithField("root", tree.Root()).Info("Succeeded to write file")
}
