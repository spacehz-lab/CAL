package probe

import (
	"os"
	"path/filepath"
	"strconv"
)

const probesDir = "probes"

func buildTargets(req *Request) ([]Target, error) {
	plansByIndex := make(map[int]int, len(req.Plans))
	for index := range req.Plans {
		candidateIndex := req.Plans[index].CandidateIndex
		if candidateIndex < 0 || candidateIndex >= len(req.Candidates) {
			return nil, newError(CodeInvalidProbeInput, "probe plan candidate index is out of range")
		}
		if _, ok := plansByIndex[candidateIndex]; ok {
			return nil, newError(CodeInvalidProbeInput, "duplicate probe plan candidate index")
		}
		plansByIndex[candidateIndex] = index
	}
	targets := make([]Target, len(req.Candidates))
	for index := range req.Candidates {
		planIndex, ok := plansByIndex[index]
		if !ok {
			return nil, newError(CodeInvalidProbeInput, "candidate is missing probe plan")
		}
		targets[index] = Target{
			CandidateIndex: index,
			Candidate:      &req.Candidates[index],
			Plan:           &req.Plans[planIndex],
			WorkDir:        candidateWorkDir(req.WorkRoot, index),
		}
	}
	return targets, nil
}

func candidateWorkDir(root string, candidateIndex int) string {
	return filepath.Join(filepath.Clean(root), probesDir, strconv.Itoa(candidateIndex))
}

func prepareWorkDir(path string, keep bool) (func(), error) {
	if path == "" {
		return func() {}, newError(CodeInvalidProbeInput, "probe workdir is required")
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return func() {}, wrapError(CodeProbeMaterializeFailed, "create probe workdir", err)
	}
	if keep {
		return func() {}, nil
	}
	return func() { _ = os.RemoveAll(path) }, nil
}
