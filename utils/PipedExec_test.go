package utils

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPipedExec_Basics(t *testing.T) {

	// echo
	{
		err := new(PipedExec).
			Command("echo", "hello").
			Run(os.Stdout, os.Stdout)
		assert.Nil(t, err)
	}

	// echo hello2 | grep hello2
	{
		err := new(PipedExec).
			Command("echo", "hello2").
			Command("grep", "hello2").
			Run(os.Stdout, os.Stdout)
		assert.Nil(t, err)
	}

	// echo hi | grep hello
	{
		err := new(PipedExec).
			Command("echo", "hi").
			Command("grep", "hello").
			Run(os.Stdout, os.Stdout)
		assert.NotNil(t, err)
	}

	// echo hi | grep hi | echo good
	{
		err := new(PipedExec).
			Command("echo", "hi").
			Command("grep", "hi").
			Command("echo", "good").
			Run(os.Stdout, os.Stdout)
		assert.Nil(t, err)
	}

	// ls at "/""
	{
		err := new(PipedExec).
			Command("ls").Wd("/").
			Run(os.Stdout, os.Stdout)
		assert.Nil(t, err)
	}

}

// Working directory
func TestPipedExec_Wd(t *testing.T) {

	/* Create structure
	tmpDir
	  tmpDir1
	  	1.txt
	  tmpDir2
	  	2.txt
	*/

	tmpDir, err := ioutil.TempDir("", "Wd")
	assert.Nil(t, err)
	defer os.RemoveAll(tmpDir)

	tmpDir1, err := ioutil.TempDir(tmpDir, "Wd")
	assert.Nil(t, err)

	tmpDir2, err := ioutil.TempDir(tmpDir, "Wd")
	assert.Nil(t, err)

	ioutil.WriteFile(filepath.Join(tmpDir1, "1.txt"), []byte("11.txt"), 0644)
	ioutil.WriteFile(filepath.Join(tmpDir2, "2.txt"), []byte("21.txt"), 0644)

	// Run ls commands

	err = new(PipedExec).
		Command("ls", "1.txt").Wd(tmpDir1).
		Run(os.Stdout, os.Stdout)
	assert.Nil(t, err)

	err = new(PipedExec).
		Command("ls", "2.txt").Wd(tmpDir2).
		Run(os.Stdout, os.Stdout)
	assert.Nil(t, err)

	err = new(PipedExec).
		Command("ls", "1.txt").Wd(tmpDir2).
		Run(os.Stdout, os.Stdout)
	assert.NotNil(t, err)

	err = new(PipedExec).
		Command("ls", "2.txt").Wd(tmpDir1).
		Run(os.Stdout, os.Stdout)
	assert.NotNil(t, err)

}

func TestPipedExec_PipeFall(t *testing.T) {

	// echo hi | grep hi | echo good => OK
	{
		err := new(PipedExec).
			Command("echo", "hi").
			Command("grep", "hi").
			Command("echo", "good").
			Run(os.Stdout, os.Stdout)
		log.Println("***", err)
		assert.Nil(t, err)
	}

	// echo hi | grep hello | echo good => FAIL
	{
		err := new(PipedExec).
			Command("echo", "hi").
			Command("grep", "hello").
			Command("echo", "good").
			Run(os.Stdout, os.Stdout)
		log.Println("***", err)
		assert.NotNil(t, err)
	}
}

func TestPipedExec_WrongCommand(t *testing.T) {
	err := new(PipedExec).
		Command("qqqqqqjkljlj", "hello").
		Run(os.Stdout, os.Stdout)
	assert.NotNil(t, err)
	log.Println(err)
}
