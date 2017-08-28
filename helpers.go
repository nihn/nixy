package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)


type ExtendedPortDefinitions struct {
	*PortDefinitions
	*App
	PortId int
	AppId string
	UpstreamName string
}

func newExtendedPortDefinitions (pd *PortDefinitions, portId int, app *App, appId string) ExtendedPortDefinitions {
	upstreamName := getUpstreamName(appId, pd.Port)
	return ExtendedPortDefinitions{pd, app, portId, appId, upstreamName}
}

func (e ExtendedPortDefinitions) IsEnabled(option string) bool {
	option = "NIXY_" + strings.ToUpper(option)

	if val, ok := e.PortDefinitions.Labels[option]; ok {
		return val == "1"
	} else if  val, ok := e.App.Labels[option]; ok {
		return val == "1"
	}
	return false
}

func (e ExtendedPortDefinitions) GetLabel(option string) string {
	option = "NIXY_" + strings.ToUpper(option)

	if val, ok := e.PortDefinitions.Labels[option]; ok {
		return val
	}
	return e.App.Labels[option]
}

func labelEnabled(pd PortDefinitions, name string) bool {
	return pd.Labels[name] == "1"
}

func normalizeAppId(appId string) string {
	return strings.Join(strings.Split(appId, "/"), "_")
}

func getUpstreamName(appId string, port int64) string {
	normalizedAppId := normalizeAppId(appId)
	return fmt.Sprintf("%s_%d", normalizedAppId, port)
}

func getUpstreams(app App, portIndex int) string {
	var buffer bytes.Buffer

	for _, task := range app.Tasks {
		buffer.WriteString(fmt.Sprintf("server %s:%d;\n",
			task.Host,task.Ports[portIndex]))
	}
	return buffer.String()
}

func iteratePortsDefinitions() <- chan ExtendedPortDefinitions {
	ch := make(chan ExtendedPortDefinitions)
	go func () {
		for appId, app := range config.Apps {
			for portId, pd := range app.PortDefinitions {
				ch <- newExtendedPortDefinitions(&pd, portId, &app, appId)
			}
		}
	}();
	return ch
}

func normalizeAppIdToStatsdMetric(appId string) string {
	metric := strings.SplitN(appId[1:], "/", 3)
	metric[len(metric)-1] = strings.Replace(metric[len(metric)-1], "/", "_", -1)
	return strings.Join(metric, ".")
}

func GCD(a, b int) int {
	for a != b {
		if a > b {
			a -= b
		} else {
			b -= a
		}
	}
	return a
}

type Weight struct {
    Task, Extra int
}

func calculateWeights(app App, labels map[string]string) Weight {
	extra_upstreams := strings.Split(labels["NIXY_EXTRA_UPSTREAMS"], " ")
	extra_weight, err := strconv.ParseFloat(labels["NIXY_EXTRA_UPSTREAMS_WEIGHT"], 64)
	if err != nil || extra_weight < 0.001 || extra_weight > 0.999 {
		return Weight{1, 1}
	}
	current_tasks := float64(len(app.Tasks)) / float64(len(app.Tasks) + len(extra_upstreams))
	current_extra := 1 - current_tasks
	extra_task_weight := int(1000 * extra_weight/current_extra)
	task_weight := int(1000 * (1-extra_weight)/current_tasks)
	if task_weight == 0.0 || extra_weight == 0.0 {
		return Weight{task_weight, extra_task_weight}
	}
	gcd := GCD(task_weight, extra_task_weight)
	return Weight{task_weight /gcd, extra_task_weight / gcd}
}
