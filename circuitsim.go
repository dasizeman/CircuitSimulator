package main

import (
    "fmt"
    "os"
    "bufio"
    "log"
    "strings"
    "strconv"
    "time"
)


type Component struct {

    typeName       string
    name           string
    logic          func([]bool)[]bool
    handler        func(*Component,int)
    outputs        []chan bool
    inputs         []*chan bool
    terminals      []*chan bool
    isTerminal     bool

    // Clock specific
    clkOuts        []chan bool
    clkSync        chan bool
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

func dFlipFlopHandler(comp *Component, arg int) {
    var Q, init bool = true, true
    if arg == 1 {
        Q = true
    } else {
        Q = false
    }
    for {
        // Wait for a clock signal (input 0 by convention)
        clk := <-*(comp.inputs[0])


        // Don't try get an input on the first cycle, just output the initial
        // value
        if !init {
            // Output Q and !Q (outputs 0 and 1)
            // If rising edge, sample D input (input 1 by convention)
            if clk {
                Q = <-*(comp.inputs[1])
            } else {
                <-*(comp.inputs[1])
            }
        }
        init = false

        invals := []bool{Q}
        outvals := comp.logic(invals)
        for i,_ := range(comp.outputs) {
            outchan := &(comp.outputs[i])
            for j := 0; j < cap(*outchan); j++ {
                (*outchan) <- outvals[i]
            }
        }

    }
}

func clkHandler(comp *Component, arg int) {
    clk := true

    // Hz
    interval := time.Second / time.Duration(arg)

    for {
        // Send clock signals
        for i,_ := range(comp.clkOuts) {
            comp.clkOuts[i] <- clk
        }
        clk = !clk

        // Wait to be notified that the last propogation is complete
        <-comp.clkSync

        // Wait for the delay based on the frequency
        <-time.After(interval)
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
        case "dff":
            newComponent.logic = DFF
            newComponent.handler = dFlipFlopHandler
            numInputs = 2
            numOutputs = 2
        case "clk":
            newComponent.handler = clkHandler
            numInputs = 0
            numOutputs = 1


        default:
            log.Fatalf("Unrecognized component type name: %s", componentType)
        }

        newComponent.inputs         = make([]*chan bool, numInputs)
        newComponent.outputs        = make([]chan bool, numOutputs)
        newComponent.clkSync        = make(chan bool)

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

    isClk := (comp.typeName == "clk")

    outputIdx := 0
    currentOutput := &(comp.outputs[outputIdx])
    numConnections := 0

    tokens := strings.Split(line, " ")

    if (len(tokens) < 3) {
        log.Fatalf("Component %d: Invalid component specification", componentIdx)
    }

    tokenIdx := 1
    firstOutIdx := 1
    if components[componentIdx].typeName == "source" ||
        components[componentIdx].typeName == "dff"{
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
            outputIdx++
            if outputIdx > len(comp.outputs) - 1 {
                log.Fatalf("Component %d: Too many outputs specified", componentIdx)
            }

            *(currentOutput) = make(chan bool, numConnections)
            numConnections = 0

            currentOutput = &(comp.outputs[outputIdx])
            tokenIdx++
            continue
        }

        // "res" specifies the value on this output is a result
        if token == "res" {
            numConnections++
            comp.terminals = append(comp.terminals, currentOutput)
            comp.isTerminal = true

        } else {
            // Check if we have two numbers, representing component and
            // component input indices.  First check if this is a valid number
            outputCompIdx, err := strconv.Atoi(token)
            if err != nil {
                log.Fatalf("Component %d: Invalid connection component index specification %s (must be a number)",
                    componentIdx, token)
            }
            if outputCompIdx >= len(components) {
                log.Fatalf("Component %d: Connection component index %d out of bounds", componentIdx, outputCompIdx)
            }


            // Next check if we have an input index
            if (tokenIdx + 1) >= len(tokens) {
                log.Fatalf("Component %d: no input number specified for connection component index %d",
                    componentIdx, outputCompIdx)
            }

            // ... and if its valid
            tokenIdx++
            inputIdx, err := strconv.Atoi(tokens[tokenIdx])
            if err != nil {
                log.Fatalf("Component %d: invalid input sprecification for connection component index %d (must be a number)",
                componentIdx, outputCompIdx)
            }

            if inputIdx >= len(components[outputCompIdx].inputs) {
                log.Fatalf("Component %d: input index %d out of bounds", componentIdx, inputIdx)
            }

            // Finally we can make this connection by setting the appropriate
            // component's input channel pointer

            // A clock is special, it only really has one output, and this uses
            // unbuffered channels to ensure synchronization
            var outputToConnect *chan bool
            if isClk {
                comp.clkOuts = append(comp.clkOuts, make(chan bool))
                outputToConnect = &comp.clkOuts[len(comp.clkOuts)-1]
            } else {
                outputToConnect = currentOutput
            }

            components[outputCompIdx].inputs[inputIdx] = outputToConnect

            numConnections++
        }

        tokenIdx++
    }

    // Update info for the final output
    *(currentOutput) = make(chan bool, numConnections)

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

func readBinaryInputAsInt(scanner *bufio.Scanner) (res int) {
    scanner.Scan()
    input := scanner.Text()
    for input != "0" && input != "1" {
        fmt.Printf("Please enter 0 or 1.\n")
        scanner.Scan()
        input = scanner.Text()
    }

    res, _ = strconv.Atoi(input)
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
func DFF(in []bool) (out []bool) {
    out = append(out, in[0])
    out = append(out, !in[0])
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

    var clk *Component
    // Kick off components
    for i,_ := range(components) {
        componentPtr := &components[i]

        var arg int
        switch componentPtr.typeName {

        case "source":
            fmt.Printf("[%s] Source value: ", componentPtr.name)
            arg = readBinaryInputAsInt(scanner)

            //fmt.Printf("Starting source routine\n")

        case "dff":
            fmt.Printf("[%s] DFF value: ", componentPtr.name)
            arg = readBinaryInputAsInt(scanner)

        case "clk":
            clk = componentPtr
            fmt.Printf("Clock frequency: ")
            scanner.Scan()
            hertz, err := strconv.Atoi(scanner.Text())
            if err != nil {
                log.Fatal(err)
            }
            arg = hertz

        default:
            //fmt.Printf("Starting component routine\n")
            arg = 0

        }

        go componentPtr.handler(componentPtr,arg)
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
            fmt.Printf("[Output] %08b\r", outNum)
            lastValue = outNum
        }

        // If we have a clock, notify it that the circuit propogation is
        // complete
        if clk != nil {
            clk.clkSync <- true
        }
    }
}


