package main

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	gc "github.com/untillpro/gochips"
)

const TESTFOLDERNAME = "TestQS"
const MAINFOLDER = "MAIN"
const FOLDER1 = "F1"
const FOLDER2 = "F2"
const TESTFILENAME = "txtFile"
const QSUCMD = "qsu.cmd"
const QSUCMDTXT = "yes | qs u"

var basepath string = ""

func TestQSgs(t *testing.T) {
	// Tests command "GS"
	fmt.Println("\n1. Build qs utility test begins")

	basepath, _ = os.Getwd()
	err := new(gc.PipedExec).
		Command("go", "build").
		Run(os.Stdout, os.Stdout)

	assert.False(t, err != nil, "Something went wrong. See error message above")
	fmt.Println("\n1. Build qs utility test completed")
}

func TestQSUD(t *testing.T) {
	// Prepare GIT repositories and test data
	initTestData(t)

	// Test Add file with Push and Pull
	qSUDAdd(t)

	// Test Delete file with Push and Pull
	qSDel(t)

	// Delete created structure
	deinitTestData(t)
}

func qSUDAdd(t *testing.T) {

	fmt.Println("\n2. Test for Add file begins")
	// Tests command "U"

	// Create test text file 'test_file_name.text' exists, delete it
	sfname := path.Join(TESTFOLDERNAME, FOLDER1, TESTFILENAME)
	f, err := os.Create(sfname)
	f.Close()
	assert.False(t, err != nil, "Something went wrong with creating test file file. See error message above")

	os.Chdir(path.Join(TESTFOLDERNAME, FOLDER1))
	fmt.Println("Test for command 'U' begins")
	err = new(gc.PipedExec).
		Command(QSUCMD).
		Run(os.Stdout, os.Stdout)
	assert.False(t, err != nil, "Something went wrong with 'qs U'. See error message above")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("Test for command 'U' finished")

	fmt.Println("Test for command 'D' begins")
	os.Chdir(path.Join(basepath, TESTFOLDERNAME, FOLDER2))

	err = new(gc.PipedExec).
		Command("qs", "d").
		Run(os.Stdout, os.Stdout)
	assert.False(t, err != nil, "Something went wrong with 'D'. See error message above")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("Test for command 'D' finished")

	// create file if not exists
	if !FlExists(path.Join(basepath, TESTFOLDERNAME, FOLDER2, TESTFILENAME)) {
		t.Errorf("TxtFile in F2 does not exist")
		return
	}
	fmt.Println("\nTxtFile in F2 Exists! It's all right!")
	fmt.Println("\n2. Test for Add file completed")
}

func qSDel(t *testing.T) {
	fmt.Println("\n3. Test Delete file begins")

	// Delete created test file
	os.Remove(path.Join(basepath, TESTFOLDERNAME, FOLDER2, TESTFILENAME))

	fmt.Println("Test for command 'U' begins")
	os.Chdir(path.Join(basepath, TESTFOLDERNAME, FOLDER2))
	err := new(gc.PipedExec).
		Command(QSUCMD).
		Run(os.Stdout, os.Stdout)
	assert.False(t, err != nil, "Something went wrong with 'qs U'. See error message above")
	fmt.Println("Test for command 'U' finished")

	os.Chdir(path.Join(basepath, TESTFOLDERNAME, FOLDER1))
	fmt.Println("Test for command 'D' begins")
	err = new(gc.PipedExec).
		Command("qs", "d").
		Run(os.Stdout, os.Stdout)
	assert.False(t, err != nil, "Something went wrong with 'D'. See error message above")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("Test for command 'D' finished")

	// create file if not exists
	if FlExists(path.Join(basepath, TESTFOLDERNAME, FOLDER1, TESTFILENAME)) {
		t.Errorf("TxtFile in F1 still exists")
		return
	}
	fmt.Println("\n3. Test Delete file completed")
}

func initTestData(t *testing.T) {
	fmt.Println("\nInit test data")

	// Remove Test folder
	os.RemoveAll(TESTFOLDERNAME)

	// Create test folder
	os.MkdirAll(TESTFOLDERNAME, os.ModePerm)

	// Init bare git repo
	err := new(gc.PipedExec).
		Command("git", "init", "--bare", path.Join(TESTFOLDERNAME, MAINFOLDER)).
		Run(os.Stdout, os.Stdout)
	assert.False(t, err != nil, "Something went wrong with git init. See error message above")

	// Clone origin repo to test1
	err = new(gc.PipedExec).
		Command("git", "clone", path.Join(TESTFOLDERNAME, MAINFOLDER), path.Join(TESTFOLDERNAME, FOLDER1)).
		Run(os.Stdout, os.Stdout)
	assert.False(t, err != nil, "Something went wrong with clone 1. See error message above")

	// Create cmd file with text "yes | qs u"
	sfname := path.Join(TESTFOLDERNAME, FOLDER1, QSUCMD)
	f, err := os.Create(sfname)
	assert.False(t, err != nil, "Something went wrong with Creating CMD file. See error message above")
	defer f.Close()
	d2 := []byte(QSUCMDTXT)
	_, err = f.Write(d2)
	assert.False(t, err != nil, "Something went wrong when write CMD file content. See error message above")

	// Clone origin repo to test2
	err = new(gc.PipedExec).
		Command("git", "clone", path.Join(TESTFOLDERNAME, MAINFOLDER), path.Join(TESTFOLDERNAME, FOLDER2)).
		Run(os.Stdout, os.Stdout)
	assert.False(t, err != nil, "Something went wrong with clone 2. See error message above")
}
func FlExists(name string) bool {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

func deinitTestData(t *testing.T) {
	// delete test folder after test
	os.Chdir(basepath)
	os.RemoveAll(TESTFOLDERNAME)
}

func TestDeleteDup(t *testing.T) {
	str := deleteDupMinus("13427-Show--must----go---on")
	assert.Equal(t, str, "13427-Show-must-go-on")
	str = deleteDupMinus("----Show--must----")
	assert.Equal(t, str, "-Show-must-")
}

func TestGetTaskIDFromUrl(t *testing.T) {
	topicid := getTaskIDFromURL("https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, topicid, "13427")
	topicid = getTaskIDFromURL("https://dev.heeus.io/launchpad/13428")
	assert.Equal(t, topicid, "13428")
	topicid = getTaskIDFromURL("13429")
	assert.Equal(t, topicid, "13429")
}

func TestGetBranchName(t *testing.T) {
	str := getBranchName("Show", "must", "go", "on", "https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, str, "13427-Show-must-go-on")
	str = getBranchName("Show   ivv?", "must    ", "go", "on---", "https://dev.heeus.io/launchpad/#!13427")
	assert.Equal(t, str, "13427-Show-ivv-must-go-on")
	str = getBranchName("Show", "must", "go", "on")
	assert.Equal(t, str, "Show-must-go-on")
	str = getBranchName("Show")
	assert.Equal(t, str, "Show")
}
