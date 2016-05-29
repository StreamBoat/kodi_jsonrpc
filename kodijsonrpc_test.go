package kodijsonrpc

import "fmt"

func ExampleNew() {
	kodi, err := New(`localhost:9090`, 15) // timeout after 15 secs
	defer kodi.Close()                     // always close to free resources

	if err != nil {
		panic(fmt.Sprintf(`Couldn't connect to Kodi: %v`, err))
	}

	request := Request{Method: `JSONRPC.Version`}
	response, err := kodi.Send(request, true) // second param says we need a response

	if err != nil {
		panic(fmt.Sprintf(`Kodi send failed with error: %v`, err))
	}

	// wait indefinitely for response (timeout 0)
	result, err := response.Read(0)

	if err != nil {
		panic(fmt.Sprintf(`Kodi responded with error: %v`, err))
	}

	fmt.Println(result)

	// Output:
	// map[version:map[major:6 minor:14 patch:3]]
}
