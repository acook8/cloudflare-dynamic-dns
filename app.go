package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

type Response struct {
	Result		[]Zone		`json:"result"`
}

type Zone struct {
	Id		string		`json:"id"`
}

type ResponseDNS struct {
	Result		[]DNS		`json:"result"`
	Success		bool		`json:"success"`
	Errors		[]Errors	`json:"errors"`
}

type DNS struct {
	Ip			string		`json:"content"`
	Id			string		`json:"id"`
}

type Errors struct {
	Code		int			`json:"code"`
	Message		string		`json:"message"`
}

// Get Public IP
func getPublicIP() string {
	response, err := http.Get("https://api.ipify.org")

	if err != nil {
		fmt.Print(err.Error())
		os.Exit(1)
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	ip := string(responseData)
	return ip
}

//Get Zone ID
func getZoneId() string {
	apiKey := os.Getenv("CF_API_KEY")
	zone := os.Getenv("ZONE")
	url := "https://api.cloudflare.com/client/v4/" + "zones?name=" + zone
	client := &http.Client {
  	}
	req, err:= http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
	  }
	req.Header.Add("Authorization", "Bearer " + apiKey)
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}

	var responseObject Response
	json.Unmarshal(body, &responseObject)
	var zoneId Zone = responseObject.Result[0]
	id := zoneId.Id
	return id
}
//Check if DNS record exists, return IP if it does, return 0 if it doesn't
func getDNSRecord(zoneId string) DNS {
	apiKey := os.Getenv("CF_API_KEY")
	subdomain := os.Getenv("SUBDOMAIN")
	zone := os.Getenv("ZONE")

	url := "https://api.cloudflare.com/client/v4/zones/" + zoneId + "/dns_records?type=A&name=" + subdomain + "." + zone + "&page=1&per_page=100&order=type&direction=desc&match=all"

	client := &http.Client {
	}
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		fmt.Println(err)
	}
	req.Header.Add("Authorization", "Bearer " + apiKey)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
	}
	var responseObject ResponseDNS
	json.Unmarshal(body, &responseObject)
	resultLength := len(responseObject.Result)
	if resultLength == 0 {
		var dns DNS
		dns.Ip = "0"
		return dns
	}
	var dns DNS = responseObject.Result[0]
	return dns
}

//Create DNS Record
func createDNSRecord(zoneId string, dynamicIp string)  {
	apiKey := os.Getenv("CF_API_KEY")
	subdomain := os.Getenv("SUBDOMAIN")
	zone := os.Getenv("ZONE")
	dnsEntry := subdomain + "." + zone
	url := "https://api.cloudflare.com/client/v4/zones/" + zoneId + "/dns_records"

	payload := strings.NewReader(`{"type":"A","name":"`+ dnsEntry + `","content":"` + dynamicIp + `","ttl":3600,"priority":10,"proxied":true}`)

	client := &http.Client {
	}
	req, err := http.NewRequest("POST", url, payload)

	if err != nil {
	fmt.Println(err)
	return
	}
	req.Header.Add("Authorization", "Bearer " + apiKey)
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
	fmt.Println(err)
	return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
	fmt.Println(err)
	return
	}
	var responseObject ResponseDNS
	json.Unmarshal(body, &responseObject)
	success := responseObject.Success
	if !success {
		fmt.Println("An error has occurred on the dns creation")
		for i := 0; i < len(responseObject.Errors); i++ {
			var errorObject Errors = responseObject.Errors[i]
			fmt.Printf("Code: %v\n", errorObject.Code)
			fmt.Println("Message: " + errorObject.Message)
		}
	}
	if success {
		fmt.Println("Created DNS record")
	}
}

//Update DNS Record
func patchDNSRecord(zoneId string, dynamicIp string, dnsId string)  {
	apiKey := os.Getenv("CF_API_KEY")
	subdomain := os.Getenv("SUBDOMAIN")
	zone := os.Getenv("ZONE")
	dnsEntry := subdomain + "." + zone
	url := "https://api.cloudflare.com/client/v4/zones/" + zoneId + "/dns_records/" + dnsId

	payload := strings.NewReader(`{"type":"A","name":"`+ dnsEntry + `","content":"` + dynamicIp + `"}`)

	client := &http.Client {
	}
	req, err := http.NewRequest("PATCH", url, payload)

	if err != nil {
	fmt.Println(err)
	return
	}
	req.Header.Add("Authorization", "Bearer " + apiKey)
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
	fmt.Println(err)
	return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
	fmt.Println(err)
	return
	}
	var responseObject ResponseDNS
	json.Unmarshal(body, &responseObject)
	success := responseObject.Success
	if !success {
		fmt.Println("An error has occurred on the dns update")
		for i := 0; i < len(responseObject.Errors); i++ {
			var errorObject Errors = responseObject.Errors[i]
			fmt.Printf("Code: %v\n", errorObject.Code)
			fmt.Println("Message: " + errorObject.Message)
		}
	}
	if success {
		fmt.Println("Updated DNS record")
	}
}

func main()  {

	publicIp := getPublicIP()
	fmt.Println("public ip: " + publicIp)

	zoneId := getZoneId()

	//Get subdomain IP
	dnsRecord := getDNSRecord(zoneId)

	//create dns record if it doesn't exist
	if dnsRecord.Ip == "0" {
		createDNSRecord(zoneId, publicIp)
	}

	//update dns record if it exists and doesn't match public ip
	if dnsRecord.Ip != publicIp && dnsRecord.Ip != "0" {
		patchDNSRecord(zoneId, publicIp, dnsRecord.Id)
	}





	/* TODO:
	[x] Get Zone ID
	[x] Get current DNS
	[x] Compare it to IP
	[x] Update if neccessary
	*/
}