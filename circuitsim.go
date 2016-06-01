package main

import (
    "fmt"
    "os"
    "bufio"
    "log"
    "strings"
    "strconv"
)


type Component struct {

    // A string for the type of component
    typeName       string

    // A unique name for the component
    name           string

    // A function to perorm gate logic on the inputs and produce and output
    logic          func([]bool)[]bool

    // A function with handling code unique to the component
    handler        func(*Component,int)

    // Channels to send output values to
    outputs        []chan bool

    // A channel to notify components reading from outputs that all fanned
    // outputs have gotten their value
    fanSyncOut     []chan bool

    // A channel to wait on after taking a value from an input
    fanSyncIn     []*chan bool

    // Channels to recieve input values from
    inputs         []*chan bool

    // Outputs that should be read as results
    terminals      []*chan bool

    // Does this component contain any terminal outputs
    isTerminal     bool
}

func basicHandler(comp *Component, arg int) {
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

func sourceHandler(comp *Component, arg int) {
    // A source just blasts the output value to fill the capacity of its output
    // channel (aka to all connected inputs)
    for {
        var output bool
        if arg == 0 {
            output = false
        } else {
            output = true
        }

        outchan := &(comp.outputs[0])
        for i := 0; i < cap(*outchan); i++ {
            (*outchan) <- output
        }
    }
}


func parseFile(filepath string) (components         []Component,
                                 terminalComponents []*Component) {
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

        // Ignore lines starting with / to allow comments
        if line[0] == '/' {
            continue
        }
        componentType := strings.ToLower(strings.Split(line," ")[0])

        newComponent := Component{}
        newComponent.typeName = componentType
        numInputs := 0
        numOutputs := 0


        switch componentType {
        case "not":
            newComponent.logic = NOT
            newComponent.handler = basicHandler
            numInputs = 1
            numOutputs = 1

        case "and":
            newComponent.logic = AND
            newComponent.handler = basicHandler
            numInputs = 2
            numOutputs = 1

        case "or":
            newComponent.logic = OR
            newComponent.handler = basicHandler
            numInputs = 2
            numOutputs = 1

        case "nand":
            newComponent.logic = NAND
            newComponent.handler = basicHandler
            numInputs = 2
            numOutputs = 1

        case "nor":
            newComponent.logic = NOR
            newComponent.handler = basicHandler
            numInputs = 2
            numOutputs = 1

        case "xor":
            newComponent.logic = XOR
            newComponent.handler = basicHandler
            numInputs = 2
            numOutputs = 1
        case "source":
            newComponent.handler = sourceHandler
            numInputs = 0
            numOutputs = 1

        default:
            log.Fatalf("Unrecognized component type name: %s", componentType)
        }

        newComponent.inputs         = make([]*chan bool, numInputs)
        newComponent.outputs        = make([]chan bool, numOutputs)
        newComponent.fanSyncIn      = make([]*chan bool, numInputs)
        newComponent.fanSyncOut     = make([]chan bool,  numOutputs)

        components = append(components, newComponent)
        componentLines = append(componentLines, line)
    }

    // Do another pass to populate and "connect" the components

    for idx,line := range(componentLines) {
        parseComponent(line, components, idx)
        if components[idx].isTerminal {
            terminalComponents = append(terminalComponents, &components[idx])
        }
    }

    if len(components) == 0 {
        log.Fatal("No components specified")
    }

    return
}

func parseComponent(line string, components []Component, componentIdx int) {

    comp := &components[componentIdx]

    outputIdx := 0
    currentOutput := &(comp.outputs[outputIdx])
    currentOutputFanSync := &(comp.fanSyncOut[outputIdx])
    numConnections := 0

    tokens := strings.Split(line, " ")

    if (len(tokens) < 3) {
        log.Fatalf("Component %d: Invalid component specification", componentIdx)
    }

    tokenIdx := 1
    firstOutIdx := 1
    if components[componentIdx].typeName == "source" {
        firstOutIdx++
        components[componentIdx].name = tokens[tokenIdx]
    }

    if strings.ToLower(tokens[firstOutIdx]) != "out" {
        log.Fatalf("Component %d: A component must specify at least one output", componentIdx)
    }

    tokenIdx += firstOutIdx
    for tokenIdx < len(tokens) {
        token := strings.ToLower(tokens[tokenIdx])

        // We have reached the connection spec for another output
        if token == "out" {
            *(currentOutput) = make(chan bool, numConnections)
            *(currentOutputFanSync) = make(chan bool)
            numConnections = 0

            outputIdx++
            if outputIdx > len(comp.outputs) - 1 {
                log.Fatalf("Component %d: Too many outputs specified", componentIdx)
            }
            currentOutput = &(comp.outputs[outputIdx])
            currentOutputFanSync = &(comp.fanSyncOut[outputIdx])
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
            if compIdx >= len(components) {
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

            if inputIdx >= len(components[compIdx].inputs) {
                log.Fatalf("Component %d: input index %d out of bounds", componentIdx, inputIdx)
            }

            // Finally we can make this connection by setting the appropriate
            // component's input channel pointer
            components[compIdx].inputs[inputIdx] = currentOutput
            components[compIdx].fanSyncIn[inputIdx] = currentOutputFanSync

            numConnections++
        }

        tokenIdx++
    }

    // Update info for the final output
    *(currentOutput) = make(chan bool, numConnections)
    *(currentOutputFanSync) = make(chan bool)

}

func parseOutputs (bools []bool) (result int) {
    for _,v := range(bools) {
        var digit int
        if v {
            digit = 1
        } else {
            digit = 0
        }
        result = (result << 1) + digit
    }

    return
}

func NOT(in []bool) (out []bool) {
    out[0] = !in[0]
    return
}

func AND(in []bool) (out []bool) {
    out = append(out, (in[0] && in[1]))
    return
}

func OR(in []bool) (out []bool) {
    out = append(out, (in[0] || in[1]))
    return
}

func NAND(in []bool) (out []bool) {
    out = append(out,!AND(in)[0])
    return
}

func NOR(in []bool) (out []bool) {
    out = append(out, !OR(in)[0])
    return
}

func XOR(in []bool) (out []bool) {
    out = append(out, !(in[0] == in[1]))
    return
}

func main() {
    scanner := bufio.NewScanner(os.Stdin)
    fmt.Printf("Enter the path to your circuit file:\n")
    scanner.Scan()
    path := scanner.Text()

    components, terminalComponents := parseFile(path)

    fmt.Printf("Parsed file sucessfully.\n")
    fmt.Printf("Found:\nComponents: %d\nTerminals: %d\n",
        len(components), len(terminalComponents))

    // Kick off components
    for i,_ := range(components) {
        componentPtr := &components[i]

        switch componentPtr.typeName {

        case "source":
            fmt.Printf("[%s] Source value: ", componentPtr.name)
            scanner.Scan()
            input := scanner.Text()
            for input != "0" && input != "1" {
                fmt.Printf("Please enter 0 or 1.\n")
                scanner.Scan()
                input = scanner.Text()
            }

            intval, _ := strconv.Atoi(input)

            //fmt.Printf("Starting source routine\n")
            go componentPtr.handler(componentPtr, intval)

        default:
            //fmt.Printf("Starting component routine\n")
            go componentPtr.handler(componentPtr,0)

        }
    }

    // Receive from all terminal channels
    lastValue := 0
    for {
        var outValues []bool
        for i,_ := range(terminalComponents) {
            chanPtrs := (*terminalComponents[i]).terminals
            for j,_ := range(chanPtrs) {
                chanPtr := chanPtrs[j]
                val := <-(*chanPtr)
                outValues = append(outValues, val)
            }
        }
        outNum := parseOutputs(outValues)
        if outNum != lastValue {
            fmt.Printf("%b\n", outNum)
            lastValue = outNum
        }
    }
}


