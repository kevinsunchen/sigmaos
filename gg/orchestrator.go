package gg

import (
	"io/ioutil"
	"log"
	"path"
	"strings"

	db "ulambda/debug"
	"ulambda/fslib"
	np "ulambda/ninep"
)

const (
	GG_TOP_DIR = "name/gg"
	// XXX eventually make GG dirs configurable, both here & in GG
	GG_DIR           = "name/fs/.gg"
	GG_BLOB_DIR      = GG_DIR + "/blobs"
	GG_REDUCTION_DIR = GG_DIR + "/reductions"
	//  GG_DIR = GG_TOP_DIR + ".gg"
	ORCHESTRATOR              = GG_TOP_DIR + "/orchestrator"
	UPLOADER_SUFFIX           = ".uploader"
	DOWNLOADER_SUFFIX         = ".downloader"
	EXECUTOR_SUFFIX           = ".executor"
	TARGET_WRITER_SUFFIX      = ".target-writer"
	OUTPUT_HANDLER_SUFFIX     = ".output-handler"
	THUNK_OUTPUTS_SUFFIX      = ".thunk-outputs"
	INPUT_DEPENDENCIES_SUFFIX = ".input-dependencies"
	SHEBANG_DIRECTIVE         = "#!/usr/bin/env gg-force-and-run"
)

type ExecutorLauncher interface {
	Spawn(*fslib.Attr) error
}

type OrchestratorDev struct {
	orc *Orchestrator
}

func (orcdev *OrchestratorDev) Write(off np.Toffset, data []byte) (np.Tsize, error) {
	//  t := string(data)
	//  db.DPrintf("OrchestratorDev.write %v\n", t)
	//  if strings.HasPrefix(t, "Join") {
	//    orcdev.orc.join(t[len("Join "):])
	//  } else if strings.HasPrefix(t, "Leave") {
	//    orcdev.orc.leave(t[len("Leave"):])
	//  } else if strings.HasPrefix(t, "Add") {
	//    orcdev.orc.add()
	//  } else if strings.HasPrefix(t, "Resume") {
	//    orcdev.orc.resume(t[len("Resume "):])
	//  } else {
	//    return 0, fmt.Errorf("Write: unknown command %v\n", t)
	//  }
	return np.Tsize(len(data)), nil
}

func (orcdev *OrchestratorDev) Read(off np.Toffset, n np.Tsize) ([]byte, error) {
	//  if off == 0 {
	//  s := orcdev.sd.ps()
	//return []byte(s), nil
	//}
	return nil, nil
}

func (orcdev *OrchestratorDev) Len() np.Tlength {
	return 0
}

type Orchestrator struct {
	pid          string
	cwd          string
	targets      []string
	targetHashes []string
	*fslib.FsLibSrv
}

func MakeOrchestrator(args []string, debug bool) (*Orchestrator, error) {
	log.Printf("Orchestrator: %v\n", args)
	orc := &Orchestrator{}

	orc.pid = args[0]
	orc.cwd = args[1]
	orc.targets = args[2:]
	fls, err := fslib.InitFs(ORCHESTRATOR, &OrchestratorDev{orc})
	if err != nil {
		return nil, err
	}
	orc.FsLibSrv = fls
	//  db.SetDebug(true)
	orc.Started(orc.pid)
	return orc, nil
}

func (orc *Orchestrator) Exit() {
	orc.Exiting(orc.pid, "OK")
}

func (orc *Orchestrator) Work() {
	orc.setUpDirs()
	executables := orc.getExecutableDependencies()
	exUpPids := orc.uploadExecutableDependencies(executables)
	children := []string{}
	for _, target := range orc.targets {
		// XXX handle non-thunk targets
		db.DPrintf("Spawning upload worker [%v]\n", target)
		targetHash := orc.getTargetHash(target)
		targetInputDependencies := orc.getTargetInputDependencies(targetHash)
		orc.targetHashes = append(orc.targetHashes, targetHash)
		upPids := []string{orc.spawnUploader(targetHash)}
		upPids = append(upPids, exUpPids...)
		targetInputDependencies = append(targetInputDependencies, upPids...)
		exPid := spawnExecutor(orc, targetHash, targetInputDependencies)
		child := spawnThunkOutputHandler(orc, exPid, targetHash, []string{targetHash})
		finalOutput := path.Join(
			GG_REDUCTION_DIR,
			targetHash,
		)
		log.Printf("Final output will be pointed to by: %v\n", strings.ReplaceAll(finalOutput, "name", "/mnt/9p"))
		children = append(children, child)
		orc.spawnTargetWriter(target, targetHash)
	}
}

func (orc *Orchestrator) getTargetInputDependencies(targetHash string) []string {
	dependencies := []string{}
	dependenciesFilePath := path.Join(
		orc.cwd,
		".gg",
		"blobs",
		targetHash+INPUT_DEPENDENCIES_SUFFIX,
	)
	f, err := ioutil.ReadFile(dependenciesFilePath)
	if err != nil {
		db.DPrintf("No input dependencies file for [%v]: %v\n", targetHash, err)
		return dependencies
	}
	f_trimmed := strings.TrimSpace(string(f))
	if len(f_trimmed) > 0 {
		for _, d := range strings.Split(f_trimmed, "\n") {
			dependencies = append(dependencies, d+OUTPUT_HANDLER_SUFFIX)
		}
	}
	db.DPrintf("Got input dependencies for [%v]: %v\n", targetHash, dependencies)
	return dependencies
}

func (orc *Orchestrator) writeTargets() {
	for i, target := range orc.targets {
		targetReduction := path.Join(
			GG_REDUCTION_DIR,
			orc.targetHashes[i],
		)
		f, err := orc.ReadFile(targetReduction)
		if err != nil {
			log.Fatalf("Error reading target reduction: %v\n", err)
		}
		outputHash := strings.TrimSpace(string(f))
		outputPath := path.Join(
			GG_BLOB_DIR,
			outputHash,
		)
		outputValue, err := orc.ReadFile(outputPath)
		if err != nil {
			log.Fatalf("Error reading value path: %v\n", err)
		}
		err = ioutil.WriteFile(target, outputValue, 0777)
		if err != nil {
			log.Fatalf("Error writing output file: %v\n", err)
		}
		db.DPrintf("Wrote output file [%v]\n", target)
	}
}

func (orc *Orchestrator) uploadExecutableDependencies(execs []string) []string {
	pids := []string{}
	for _, exec := range execs {
		pids = append(pids, orc.spawnUploader(exec))
	}
	return pids
}

func (orc *Orchestrator) getExecutableDependencies() []string {
	execsPath := path.Join(orc.cwd, ".gg", "blobs", "executables.txt")
	f, err := ioutil.ReadFile(execsPath)
	if err != nil {
		log.Fatalf("Error reading exec dependencies: %v\n", err)
	}
	trimmed_f := strings.TrimSpace(string(f))
	return strings.Split(trimmed_f, "\n")
}

func (orc *Orchestrator) getTargetHash(target string) string {
	// XXX support non-placeholders
	targetPath := path.Join(orc.cwd, target)
	f, err := ioutil.ReadFile(targetPath)
	contents := string(f)
	if err != nil {
		log.Fatalf("Error reading target [%v]: %v\n", target, err)
	}
	shebang := strings.Split(contents, "\n")[0]
	if shebang != SHEBANG_DIRECTIVE {
		log.Fatalf("Error: [%v] is not a placeholder [%v]", targetPath, shebang)
	}
	hash := strings.Split(contents, "\n")[1]
	return hash
}

func (orc *Orchestrator) mkdirOpt(path string) {
	_, err := orc.FsLib.Stat(path)
	if err != nil {
		db.DPrintf("Mkdir [%v]\n", path)
		// XXX Perms?
		err = orc.FsLib.Mkdir(path, np.DMDIR)
		if err != nil {
			log.Fatalf("Couldn't mkdir %v", GG_DIR)
		}
	} else {
		db.DPrintf("Already exists [%v]\n", path)
	}
}

func (orc *Orchestrator) setUpDirs() {
	orc.mkdirOpt(GG_DIR)
	orc.mkdirOpt(GG_BLOB_DIR)
}

func (orc *Orchestrator) spawnUploader(targetHash string) string {
	a := fslib.Attr{}
	a.Pid = targetHash + UPLOADER_SUFFIX
	a.Program = "./bin/fsuploader"
	a.Args = []string{
		path.Join(orc.cwd, ".gg", "blobs", targetHash),
		path.Join(GG_BLOB_DIR, targetHash),
	}
	a.Env = []string{}
	a.PairDep = []fslib.PDep{}
	a.ExitDep = nil
	err := orc.Spawn(&a)
	if err != nil {
		log.Fatalf("Error spawning upload worker [%v]: %v\n", targetHash, err)
	}
	return a.Pid
}

func (orc *Orchestrator) spawnTargetWriter(target string, targetReduction string) {
	a := fslib.Attr{}
	a.Pid = target + TARGET_WRITER_SUFFIX
	a.Program = "./bin/gg-target-writer"
	a.Args = []string{
		orc.cwd,
		target,
		targetReduction,
	}
	a.Env = []string{}
	a.PairDep = []fslib.PDep{}
	a.ExitDep = []string{
		targetReduction + OUTPUT_HANDLER_SUFFIX,
	}
	err := orc.Spawn(&a)
	if err != nil {
		log.Fatalf("Error spawning target writer [%v]: %v\n", target, err)
	}
}

func spawnExecutor(launch ExecutorLauncher, targetHash string, depPids []string) string {
	a := fslib.Attr{}
	a.Pid = targetHash + EXECUTOR_SUFFIX
	a.Program = "gg-execute"
	a.Args = []string{
		"--ninep",
		targetHash,
	}
	a.Env = []string{
		"GG_STORAGE_URI=9p://mnt/9p/fs",
		"GG_DIR=/mnt/9p/fs/.gg", // XXX Make this configurable
		"GG_NINEP=true",
		"GG_VERBOSE=1",
	}
	a.PairDep = []fslib.PDep{}
	a.ExitDep = depPids
	err := launch.Spawn(&a)
	if err != nil {
		// XXX Clean this up better with caching
		//    log.Fatalf("Error spawning executor [%v]: %v\n", targetHash, err);
	}
	return a.Pid
}

func spawnThunkOutputHandler(launch ExecutorLauncher, exPid string, thunkHash string, outputFiles []string) string {
	a := fslib.Attr{}
	a.Pid = thunkHash + OUTPUT_HANDLER_SUFFIX
	a.Program = "./bin/gg-thunk-output-handler"
	a.Args = []string{
		thunkHash,
	}
	a.Args = append(a.Args, outputFiles...)
	a.Env = []string{}
	a.PairDep = []fslib.PDep{}
	a.ExitDep = []string{
		exPid,
	}
	err := launch.Spawn(&a)
	if err != nil {
		// XXX Clean this up better with caching
		//    log.Fatalf("Error spawning output handler [%v]: %v\n", thunkHash, err);
	}
	return a.Pid
}
