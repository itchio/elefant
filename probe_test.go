package elefant_test

import (
	"runtime"
	"testing"

	"github.com/itchio/elefant"
	"github.com/itchio/wharf/eos"
	"github.com/stretchr/testify/assert"
)

func Test_NotElfFile(t *testing.T) {
	f, err := eos.Open("./testdata/hello.c")
	assert.NoError(t, err)
	defer f.Close()

	_, err = elefant.Probe(f, nil)
	assert.Error(t, err)
}

func Test_Hello32(t *testing.T) {
	f, err := eos.Open("./testdata/hello32")
	assert.NoError(t, err)
	defer f.Close()

	res, err := elefant.Probe(f, nil)
	assert.NoError(t, err)
	assert.EqualValues(t, elefant.Arch386, res.Arch)
}

func Test_Hello64(t *testing.T) {
	f, err := eos.Open("./testdata/hello64")
	assert.NoError(t, err)
	defer f.Close()

	res, err := elefant.Probe(f, nil)
	assert.NoError(t, err)
	assert.EqualValues(t, elefant.ArchAmd64, res.Arch)
}

func Test_Trace(t *testing.T) {
	if runtime.GOOS != "linux" {
		return
	}

	fullpath := "./testdata/notify-send"
	f, err := eos.Open(fullpath)
	assert.NoError(t, err)
	defer f.Close()

	res, err := elefant.Probe(f, nil)
	assert.NoError(t, err)
	assert.EqualValues(t, elefant.ArchAmd64, res.Arch)

	root, err := elefant.Trace(res, fullpath)
	assert.NoError(t, err)

	t.Logf("%s", root)
}
