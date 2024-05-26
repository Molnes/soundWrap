package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func main() {

	// parse -h flag
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-h":
			fmt.Println("Usage: soundwrap")
			fmt.Println("This program is a wrapper around wpctl that uses wofi to select the default audio sink")
			fmt.Println("It reads the output of wpctl status and displays the sinks in a wofi dmenu")
			fmt.Println("The selected sink is then set as the default sink using wpctl set-default")
			os.Exit(0)
		case "--help":
			fmt.Println("Usage: soundwrap")
			fmt.Println("This program is a wrapper around wpctl that uses wofi to select the default audio sink")
			fmt.Println("It reads the output of wpctl status and displays the sinks in a wofi dmenu")
			fmt.Println("The selected sink is then set as the default sink using wpctl set-default")
			os.Exit(0)

		default:
			fmt.Println("Invalid flag")
			os.Exit(1)
		}

	} else {

		sinks := parse_wpctl_status()
		// sort the sinks by sink id

		wofi_command := exec.Command("wofi", "--show=dmenu", "--hide-scroll", "--allow-markup", "--define=hide_search=true",
			"--location=top_right", "--width=600", "--hight=200", "--xoffset=-60")
		wofi_command.Stdin = strings.NewReader(wofiString(sinks))
		wofi_output, err := wofi_command.Output()
		if err != nil {
			log.Fatal(err)
		}
		// check if the selected sink is already selected
		selectedSink := strings.TrimSpace(string(wofi_output))
		for i, sink := range sinks {
			if strings.Contains(sink.Sink_name, selectedSink) {
				if sink.Selected {
					// if the sink is already selected, do nothing
					return
				}
				sinks[i].Selected = true
			} else {
				sinks[i].Selected = false
			}
		}

		// get the sink id of the selected sink
		var selectedSinkID int
		for _, sink := range sinks {
			if sink.Selected {
				selectedSinkID = sink.Sink_id
			}
		}

		// set the selected sink as the default sink
		setSinkCommand := exec.Command("wpctl", "set-default", strconv.Itoa(selectedSinkID))
		err = setSinkCommand.Run()
		if err != nil {
			log.Fatal(err)
		}
	}
	os.Exit(0)
}

type Sink struct {
	Sink_id   int
	Sink_name string
	Selected  bool
}

func (s Sink) String() string {
	sinkId := strconv.Itoa(s.Sink_id)
	sinkName := s.Sink_name
	selected := strconv.FormatBool(s.Selected)
	if s.Selected {
		// if the sink is selected, make the sink name bold and green
		sinkName = fmt.Sprintf("\033[1;32m%s\033[0m", s.Sink_name)
		selected = fmt.Sprintf("\033[1;32m%s\033[0m", selected)
	}
	if !s.Selected {
		// if the sink is not selected, make the sink name bold and red
		sinkName = fmt.Sprintf("\033[1;31m%s\033[0m", s.Sink_name)
		selected = fmt.Sprintf("\033[1;31m%s\033[0m", selected)
	}

	return fmt.Sprintf("Sink ID: %s, Sink Name: %s, Selected: %s", sinkId, sinkName, selected)
}

func wofiString(s []Sink) string {
	wofiString := ""
	for _, sink := range s {
		if sink.Selected {
			wofiString += fmt.Sprintf("🔊 %s\n", sink.Sink_name)
		} else {
			wofiString += fmt.Sprintf("%s\n", sink.Sink_name)
		}
	}
	return wofiString
}

func parse_wpctl_status() []Sink {
	output, err := exec.Command("wpctl", "status").Output()
	if err != nil {
		log.Fatal(err)
	}

	filteredSinks, err := parse_output(output)
	if err != nil {
		log.Fatal(err)
	}

	// read the sinks into a slice of Sink structs
	sinkArray, err := parseInto(filteredSinks)
	if err != nil {
		log.Fatal(err)
	}
	return sinkArray
}

// parse_output takes the output of the wpctl status command and returns a filtered slice of strings
func parse_output(output []byte) (filteredSinks []string, err error) {
	filtered := strings.Replace(string(output), "├", "", -1)
	filtered = strings.Replace(string(filtered), "└", "", -1)
	filtered = strings.Replace(string(filtered), "│", "", -1)
	filtered = strings.Replace(string(filtered), "─", "", -1)

	splitted := strings.Split(filtered, "\n")
	sinksIndex := 0
	for _, line := range splitted {
		if strings.Contains(line, "Sinks") {
			break
		}
		sinksIndex++
	}
	sinks := []string{}
	for i := sinksIndex + 1; i < len(splitted); i++ {
		if splitted[i] == "   " {
			break
		}
		sinks = append(sinks, splitted[i])
	}

	filteredSinks = []string{}
	for _, sink := range sinks {
		// Remove the [vol: at the end of the string
		sinkName := strings.Split(sink, "[vol:")[0]
		filteredSinks = append(filteredSinks, sinkName)
	}

	return filteredSinks, nil
}

// parseInto takes a slice of strings and returns a slice of Sink structs
func parseInto(lines []string) (sinks []Sink, err error) {
	sinkArray := []Sink{}
	for _, sink := range lines {
		// split the sink string into the sink id and the sink name
		//   *   53. Navi 21/23 HDMI/DP Audio Controller Digital Stereo (HDMI 5)
		sinkParts := strings.Split(sink, ".")
		if len(sinkParts) < 2 {
			continue
		}

		selected := false
		// if there is a * at the beginning of the string, remove it
		if strings.Contains(sinkParts[0], "*") {
			sinkParts[0] = strings.Replace(sinkParts[0], "*", "", -1)
			selected = true
		}

		// trim the sink id and sink name
		sinkParts[0] = strings.TrimSpace(sinkParts[0])
		sinkParts[1] = strings.TrimSpace(sinkParts[1])

		// convert the sink id to an integer
		sinkID, err := strconv.Atoi(sinkParts[0])
		if err != nil {
			return sinkArray, err
		}

		sinkName := strings.TrimSpace(sinkParts[1])

		// create a new Sink struct and add it to the sinkArray
		sinkArray = append(sinkArray, Sink{sinkID, sinkName, selected})
	}
	if sinkArray == nil {
		return sinkArray, fmt.Errorf("sinkArray is nil")
	}

	return sinkArray, nil
}
