package main

import (
	"container/list"
	"errors"
	"fmt"
	"net"
	"os"
	sc "strconv"
	st "strings"
)

func main() {
	// Uncomment this block to pass the first stage
	//
	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			continue
		}

		go func() {
			requestHandler(conn)
			defer conn.Close()
		}()
	}
}

func requestHandler(c net.Conn) {
	data := make([]byte, 1024)

	for {
		_, err := c.Read(data)
		if err != nil {
			fmt.Println("Error reading request connection", err.Error())
			return
		}
		fmt.Printf("Got raw command of %s\n", string(data))
		resp, err := generateResponse(data)
		if err != nil {
			fmt.Println("Failed to generate response", err.Error())
			return
		}

		_, err = c.Write(resp)
		if err != nil {
			fmt.Println("Error writing respnose", err.Error())
			return
		}

	}
}

func generateResponse(data []byte) ([]byte, error) {
	req, err := parse(string(data))
	if err != nil {
		fmt.Println("Error parsing command", err.Error())
		return nil, err
	}
	resp := make([]byte, 1024)
	switch req.(type) {
	case *string:
		reqStr, _ := req.(*string)
		if *reqStr == "PING" {
			pong := formatRESPString("PONG")
			resp = []byte(pong)
		}
	case *list.List:
		l, ok := req.(*list.List)
		if ok {
			elem := l.Front()
			if elem.Value == "ECHO" {
				arg, _ := elem.Next().Value.(string)
				echo := formatRESPString(arg)
				resp = []byte(echo)
			}
		}
	default:
		fmt.Println("Unknown result")
		return nil, errors.New("Not expected command")
	}

	return resp, nil
}

func parse(data string) (interface{}, error) {
	// fmt.Println(data[:1])
	switch data[:1] {
	case "+":
		return parseRESPString(data[1:])
	case "-":
		return parseRESPError(data[1:])
	case ":":
		return parseRESPInt(data[1:])
	case "$":
		return parseRESPBulkStr(data[1:])
	case "*":
		return parseRESPArr(data[1:])
	default:
		return nil, errors.New("Not supported type")
	}
}

func parseRESPError(str string) (*string, error) {
	return nil, nil
}

func parseRESPString(str string) (*string, error) {
	splited := st.Split(str, "\r\n")
	return &splited[0], nil
}

func parseRESPInt(str string) (*int, error) {
	splited := st.Split(str, "\r\n")
	num, err := sc.Atoi(splited[0])
	if err != nil {
		fmt.Printf("Parsing number failed %s\n", str)
		return nil, errors.New("Int - wrong format")
	}

	return &num, nil
}

func parseRESPBulkStr(str string) (*string, error) {
	splited := st.Split(str, "\r\n")
	n, err := sc.Atoi(splited[0])
	if err != nil {
		fmt.Printf("bulkstr length not in number, %s\n", str)
		return nil, errors.New("Bulk string wrong format")
	}
	result := splited[1]

	if len(result) != n {
		fmt.Printf("Bulkstr length error : %s, expected length: %d \n", result, n)
		return nil, errors.New("Error parse bulk str")
	}

	return &result, nil
}

func parseRESPArr(str string) (*list.List, error) {
	result := list.New()

	n, err := sc.Atoi(str[:1])
	if err != nil {
		fmt.Println("Arr parse failed - not a number first value", err.Error())
		return nil, err
	}

	if n == 0 {
		return result, nil
	}
	splited := st.Split(str[1:], "\r\n")
	for i := 0; i < len(splited)-1; i++ {
		rawElem := splited[i]
		if len(rawElem) == 0 {
			continue
		}
		if len(rawElem) > 0 && rawElem[:1] == "$" {
			rawElem += "\r\n" + splited[i+1]
			i++
		}
		elem, err := parse(rawElem)
		if err != nil {
			fmt.Println("Error parsing element", err.Error())
			return nil, err
		}

		elemStr, _ := elem.(*string)
		result.PushBack(*elemStr)
	}

	if result.Len() != n {
		fmt.Printf("Result list length doesn't match with given length %d, %+v\n", n, result)
		return nil, errors.New("Length error")
	}

	return result, nil
}

func formatRESPString(str string) string {
	return fmt.Sprintf("+%s\r\n", str)
}
