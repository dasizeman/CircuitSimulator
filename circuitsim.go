package main

import (
    //"fmt"
    "os"
    "bufio"
    "log"
    "strings"
    "strconv"
)


type Component struct {

    name           string
    logic          func([]bool)[]bool
    outputs        []chan bool
    inputs         []*chan bool
    terminals      []*chan bool
    isTerminal     bool
}

func (comp *Component) runBasic() {
    // Loop forever
    for {
        // Get inputs
        var invals []bool
        for _,v := range(comp.inputs) {
            val := <-(*v)
            invals = append(invals, val)
        }

        // Perform the logic, and send to the output channels
        outvals := comp.logic(invals)
        for i,_ := range(comp.outputs) {
            outchan := &(comp.outputs[i])
            for j := 0; j < cap(*outchan); j++ {
                (*outchan) <- outvals[i]
            }
        }
    }
}

func (comp *Component) runAsSource(output bool) {
    // A source just blasts the output value to fill the capacity of its output
    // channel (aka to all connected inputs)
    outchan := &(comp.outputs[0])
    for i := 0; i < cap(*outchan); i++ {
        (*outchan) <- output
    }
}


func parseFile(filepath string) (components []Component, sources []*Component, terminalComponents []*Component) {
    // Try to open the file
    file, err := os.Open(filepath)
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()

    // On first pass, create all of the components, set their names and logic
    // handlers
    var componentLines []string
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        componentType := strings.ToLower(strings.Split(line," ")[0])

        newComponent := Component{}
        newComponent.name = componentType
        numInputs := 0
        numOutputs := 0


        switch componentType {
        case "not":
            newComponent.logic = NOT
            numInputs = 1
            numOutputs = 1

        case "and":
            newComponent.logic = AND
            numInputs = 2
            numOutputs = 1

        case "or":
            newComponent.logic = OR
            numInputs = 2
            numOutputs = 1

        case "nand":
            newComponent.logic = NAND
            numInputs = 2
            numOutputs = 1

        case "nor":
            newComponent.logic = NOR
            numInputs = 2
            numOutputs = 1

        case "xor":
            newComponent.logic = XOR
            numInputs = 2
            numOutputs = 1
        case "source":
            numInputs = 0
            numOutputs = 1

        default:
            log.Fatalf("Unrecognized component name: %s", componentType)
        }

        newComponent.inputs         = make([]*chan bool, numInputs)
        newComponent.outputs        = make([]chan bool, numOutputs)

        components = append(components, newComponent)
        componentLines = append(componentLines, line)

        if componentType == "source" {
            sources = append(sources, &components[len(components)-1])
        }
    }

    // Do another pass to populate and "connect" the components

    for idx,line := range(componentLines) {
        parseComponent(line, &components, idx)
        if components[idx].isTerminal {
            terminalComponents = append(terminalComponents, &components[idx])
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

func parseComponent(line string, components *[]Component, componentIdx int) {

    comp := &(*components)[componentIdx]

    outputIdx := 0
    currentOutput := &(comp.outputs[outputIdx])
    numConnections := 0

    tokens := strings.Split(line, " ")
    if strings.ToLower(tokens[1]) != "out" {
        log.Fatal("A non-source component must specify at least one output")
    }

    // Swallow the first "out" token.  We will then be two tokens in
    tokenIdx := 2
    for tokenIdx < len(tokens) {
        token := strings.ToLower(tokens[tokenIdx])

        // We have reached the connection spec for another output
        if token == "out" {
            *(currentOutput) = make(chan bool, numConnections)
            numConnections = 0

            outputIdx++
            if outputIdx > len(comp.outputs) - 1 {
                log.Fatalf("Component %d: Too many outputs specified", componentIdx)
            }
            currentOutput = &(comp.outputs[outputIdx])
        }

        // "res" specifies the value on this output is a result
        if token == "res" {
            numConnections++
            comp.terminals = append(comp.terminals, currentOutput)
            comp.isTerminal = true

        } else {
            // Check if we have two numbers, representing component and
            // component input indices.  First check if this is a valid number
            compIdx, err := strconv.Atoi(token)
            if err != nil {
                log.Fatalf("Component %d: Invalid connection component index specification %s (must be a number)",
                    componentIdx, token)
            }
            if compIdx >= len(*(components)) {
                log.Fatalf("Component %d: Connection component index %d out of bounds", componentIdx, compIdx)
            }


            // Next check if we have an input index
            if (tokenIdx + 1) >= len(tokens) {
                log.Fatalf("Component %d: no input number specified for connection component index %d",
                    componentIdx, compIdx)
            }

            // ... and if its valid
            tokenIdx++
            inputIdx, err := strconv.Atoi(tokens[tokenIdx])
            if err != nil {
                log.Fatalf("Component %d: invalid input sprecification for connection component index %d (must be a number)",
                componentIdx, compIdx)
            }

            if inputIdx >= len((*components)[compIdx].inputs) {
                log.Fatalf("Component %d: input index %d out of bounds", componentIdx, inputIdx)
            }

            // Finally we can make this connection by setting the appropriate
            // component's input channel pointer
            (*components)[compIdx].inputs[inputIdx] = currentOutput

            numConnections++
        }

        tokenIdx++
    }

    // Update info for the final output
    *(currentOutput) = make(chan bool, numConnections)

}

func NOT(in []bool) (out []bool) {
    out[0] = !in[0]
    return
}

func AND(in []bool) (out []bool) {
    out[0] = (in[0] && in[1])
    return
}

func OR(in []bool) (out []bool) {
    out[0] = (in[0] || in[1])
    return
}

func NAND(in []bool) (out []bool) {
    out[0] = !AND(in)[0]
    return
}

func NOR(in []bool) (out []bool) {
    out[0] = !OR(in)[0]
    return
}

func XOR(in []bool) (out []bool) {
    out[0] = !(in[0] == in[1])
    return
}

func main() {
}

