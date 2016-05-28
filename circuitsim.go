package main

import (
    "fmt"
    "os"
    "bufio"
    "log"
    "strings"
)


type Component struct {

    name           string
    logic          func([]uint8)[]uint8
    outputs        []chan uint8
    inputs         []*chan int
    numOutputWires []uint
}

type Source struct {
    name    string
    output  chan uint8
    numOutputWires uint
    outputValue uint8
}

func (comp Component) run() {
}

func (source Source) run() {

}


func NOT(in []uint8) (out []uint8) {
    return
}

func AND(in1 []uint8) (out []uint8) {
    return
}

func OR(in []uint8) (out []uint8) {
    return
}

func NAND(in []uint8) (out []uint8) {
    return
}

func NOR(in []uint8) (out []uint8) {
    return
}

func XOR(in []uint8) (out []uint8) {
    return
}

func parseFile(filepath string) (components []Component, sources []Source, terminals []*Component) {
    // Try to open the file
    file, err := os.Open(filepath)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    // On first pass, create all of the components, set their names and logic
    // handlers
    var componentLines, sourceLines []string
    scanner := bufio.NewScanner(file)
    for scanner.scan() {
        line := scanner.Text()
        componentType := strings.ToLower(strings.Split(line," ")[0])

        newComponent := Component{}
        newComponent.name = componentType

        if componentType == "source" {
            newSource := Source{}
            newSource.name = componentType
            sources = append(sources, newSource)
            sourceLines = append(sourceLines, line)
        }
        else {

            switch componentType {
            case "not":
                newComponent.logic = NOT

            case "and":
                newComponent.logic = AND

            case "or"
                newComponent.logic = OR

            case "nand":
                newComponent.logic = NAND

            case "nor":
                newComponent.logic = NOR

            case "xor":
                newComponent.logic = XOR

            default:
                log.Fatalf("Unrecognized component name: %s", componentType)
            }

            components = append(components, newComponent)
            componentLines = append(componentLines, line)
        }
    }

    // Do another pass to populate and "connect" the components

    // Sources
    for idx,line := range(sourceLines) {
        parseSource(line, &sources[idx])
    }

    // Components
    for idx,line := range(componentLines) {
        parseComponent(line, &components[idx])
        if components[idx].numOutputWires == 0 {
            terminals = append(terminals, &components[idx])
        }
    }

    if len(components) == 0 {
        log.Fatal("No components specified")
    }

    if len(sources) == 0 {
        log.Fatal("No sources specified")
    }


    return
}

func parseComponent(line string, comp *Component) {

}

func parseSource(line string, source *Source) {

}

func main() {
}

