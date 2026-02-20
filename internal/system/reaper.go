package system

import (
	"sort"

	"atlas.grave/internal/model"
	"github.com/shirou/gopsutil/v3/process"
)

type Reaper struct{}

func NewReaper() *Reaper {
	return &Reaper{}
}

func (r *Reaper) Scan() ([]model.Soul, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}

	var souls []model.Soul
	for _, p := range procs {
		name, _ := p.Name()
		if name == "" { continue }
		
		cpu, _ := p.CPUPercent()
		memInfo, _ := p.MemoryInfo()
		
		var mem uint64
		if memInfo != nil {
			mem = memInfo.RSS
		}

		souls = append(souls, model.Soul{
			PID:    p.Pid,
			Name:   name,
			CPU:    cpu,
			Memory: mem,
		})
	}

	// Sort by Memory (Burden) by default
	sort.Slice(souls, func(i, j int) bool {
		return souls[i].Memory > souls[j].Memory
	})

	return souls, nil
}

func (r *Reaper) Bury(pid int32) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}
