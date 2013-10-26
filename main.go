package main

import (
	"bytes"
	"code.google.com/p/go.crypto/ssh"
	"flag"
	"fmt"
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/ec2"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	FILE_NAME = "/tmp/gopher_instances"
)

// Command Line Variables
var command string

// Flags Variables
var flagSet *flag.FlagSet

// Creation options
var instanceCount int
var instanceImage, instanceType, instanceRegionName, instanceLogin, instanceKey string

// Attack options
var numberOfRequests, concurrentRequests int
var options, url string

// EC2 Variables
var instanceRegion aws.Region

func init() {
	if len(os.Args) == 1 {
		command = "help"
	} else {
		command = os.Args[1]
	}

	flagSet = flag.NewFlagSet(command, flag.ExitOnError)

	// Creation options
	flagSet.IntVar(&instanceCount, "count", 5, "number of gophers to call into action")
	flagSet.StringVar(&instanceRegionName, "region", "us-east-1", "only us-east-1 is supported at the moment")
	flagSet.StringVar(&instanceImage, "image", "ami-2bc99d42", "only one image is supported at the moment")
	flagSet.StringVar(&instanceType, "type", "t1.micro", "specify the instance type on AWS")
	flagSet.StringVar(&instanceLogin, "login", "ubuntu", "login user for your instance")
	flagSet.StringVar(&instanceKey, "key", "gophers", "create a key-pair to push to your instance, by default it's called gophers.pem")

	// Attack options
	flagSet.IntVar(&numberOfRequests, "requests", 5, "Number of requests to perform per gopher")
	flagSet.IntVar(&concurrentRequests, "concurrent", 1, "Number of concurrent requests to make per gopher")
	flagSet.StringVar(&url, "url", "", "help message for url to attack")
	flagSet.StringVar(&options, "options", "", "additional options to pass to apache bench")

	setupRegion()
}

func main() {
	if len(os.Args) <= 1 {
		flagSet.PrintDefaults()
	} else {
		flagSet.Parse(os.Args[2:])

	}

	switch command {
	case "help":
		printInstructions()
	case "up":
		up()
	case "down":
		down()
	case "attack":
		attack()
	case "report":
		report()
	default:
		flagSet.PrintDefaults()
	}
}

func up() {
	println("Gophers are breeding")

	createInstances := ec2.RunInstances{
		MinCount:     instanceCount,
		MaxCount:     instanceCount,
		ImageId:      instanceImage,
		InstanceType: instanceType,
		KeyName:      instanceKey,
	}

	ec2Connection, err := ec2Connect()
	runInstancesResponse, err := ec2Connection.RunInstances(&createInstances)
	handleError(err)

	_, err = tagInstances(runInstancesResponse.Instances)
	handleError(err)

	for {
		instances, err := findInstances(0)
		handleError(err)

		if len(instances) == 0 {
			break
		}

		fmt.Print(".")
		time.Sleep(3 * time.Second)
	}

	instances, err := findInstances(16)
	handleError(err)

	setupResponseChannel := make(chan setupResponse)
	for _, instance := range instances {
		setupInstance(setupResponseChannel, instance.DNSName)
	}

	for i := 0; i < len(instances); i++ {
		resp := <-setupResponseChannel
		if resp.err != nil {
			fmt.Println("Looks like the gophers were inbred.")
		}
	}

	fmt.Println(fmt.Sprintf("\n%v gophers are ready to invade!", len(instances)))
}

func down() {
	fmt.Println("Terminating gophers")

	instances, err := findInstances(-1)
	handleError(err)

	instanceIds := getInstanceIds(instances)

	ec2Connection, err := ec2Connect()
	ec2Connection.TerminateInstances(instanceIds)
}

func attack() {
	fmt.Println("Gophers are on the move!")

	if url == "" {
		fmt.Println("You must provide a url to start the attack.")
		return
	}

	instances, err := findInstances(16)
	handleError(err)

	attackResponseChannel := make(chan benchmarkResponse)
	for _, instance := range instances {
		startAttack(attackResponseChannel, instance.DNSName)
	}

	var complete, failed int
	var totalTime, requestsPerSecond float32
	for i := 0; i < len(instances); i++ {
		resp := <-attackResponseChannel
		if resp.err != nil {
			fmt.Println("Looks like the gophers started a civil war.")
		}

		complete += resp.Complete
		failed += resp.Failed
		requestsPerSecond += resp.RequestsPerSecond
		totalTime += resp.TimePerRequest * float32(resp.Complete+resp.Failed)
	}

	fmt.Println("Completed requests:", complete)
	fmt.Println("Failed requests:", failed)
	fmt.Println("Requests per second:", requestsPerSecond, "[#/sec] (mean)")
	fmt.Println("Time per request:", totalTime/float32(complete+failed), "[ms] (mean)")
}

func report() {
	fmt.Println("Who know where the gophers are?")

	instances, err := findInstances(-1)
	handleError(err)

	fmt.Println("Name (DNS Name) - State")
	for _, instance := range instances {
		var instanceName string

		for _, tag := range instance.Tags {
			if tag.Key == "Name" {
				instanceName = tag.Value
			}
		}

		fmt.Println(fmt.Sprintf("%s (%s) - %s", instanceName, instance.DNSName, instance.State.Name))
	}
}

func setupInstance(response chan setupResponse, host string) {
	go func() {
		setupResponse := setupResponse{}

		client, err := sshClient(fmt.Sprintf("%s:22", host))
		if err != nil {
			setupResponse.err = append(setupResponse.err, err)
		}

		session, err := client.NewSession()
		if err != nil {
			setupResponse.err = append(setupResponse.err, err)
		}

		defer session.Close()

		var outputBuffer bytes.Buffer
		session.Stdout = &outputBuffer
		err = session.Run("sudo apt-get install apache2-utils -y")
		if err != nil {
			setupResponse.err = append(setupResponse.err, err)
		}

		response <- setupResponse
	}()
}

func startAttack(response chan benchmarkResponse, host string) {
	go func() {
		benchmarkResponse := benchmarkResponse{}

		client, err := sshClient(fmt.Sprintf("%s:22", host))
		if err != nil {
			benchmarkResponse.err = append(benchmarkResponse.err, err)
		}

		session, err := client.NewSession()
		if err != nil {
			benchmarkResponse.err = append(benchmarkResponse.err, err)
		}

		defer session.Close()

		var outputBuffer bytes.Buffer
		session.Stdout = &outputBuffer

		benchmarkCommand := fmt.Sprintf("ab -r -n %v -c %v %v \"%v\"", numberOfRequests, concurrentRequests, options, url)
		err = session.Run(benchmarkCommand)
		if err != nil {
			benchmarkResponse.err = append(benchmarkResponse.err, err)
		}

		// fmt.Println(&outputBuffer)
		outputString := outputBuffer.String()

		for _, line := range strings.Split(outputString, "\n") {
			if strings.Contains(line, "Complete requests:") {
				value := strings.TrimSpace(strings.Split(line, ":")[1])
				benchmarkResponse.Complete, _ = strconv.Atoi(value)
			}
			if strings.Contains(line, "Failed requests:") {
				value := strings.TrimSpace(strings.Split(line, ":")[1])
				benchmarkResponse.Failed, _ = strconv.Atoi(value)
			}
			if strings.Contains(line, "Requests per second:") {
				re := regexp.MustCompile("Requests per second:\\s*(\\d+.\\d+)")
				value := re.FindStringSubmatch(line)[1]
				valueFloat, _ := strconv.ParseFloat(value, 32)
				benchmarkResponse.RequestsPerSecond = float32(valueFloat)
			}
			if strings.Contains(line, "Time per request:") && !strings.Contains(line, "across all concurrent requests") {
				re := regexp.MustCompile("Time per request:\\s*(\\d+.\\d+)\\s*\\[ms\\]\\s*\\(mean\\)")
				value := re.FindStringSubmatch(line)[1]
				valueFloat, _ := strconv.ParseFloat(value, 32)
				benchmarkResponse.TimePerRequest = float32(valueFloat)
			}
		}

		response <- benchmarkResponse
	}()
}

func printInstructions() {
	fmt.Println(`gophers COMMAND [options]

gophers with grenades

A utility for arming (creating) many gophers (small EC2 instances) to attack
(load test) targets (web applications).

commands:
  up      Start a batch of load testing servers.
  attack  Begin the attack on a specific url.
  down    Shutdown and deactivate the load testing servers.
  report  Report the status of the load testing servers.
    `)
}

func ec2Connect() (*ec2.EC2, error) {
	auth, err := aws.EnvAuth()
	if err != nil {
		return nil, err
	}

	ec2Connection := ec2.New(auth, aws.USEast)

	return ec2Connection, nil
}

func setupRegion() {
	instanceRegion = aws.Regions[instanceRegionName]
}

func getInstanceIds(instances []ec2.Instance) (instanceIds []string) {
	for _, instance := range instances {
		instanceIds = append(instanceIds, instance.InstanceId)
	}

	return instanceIds
}

func tagInstances(instances []ec2.Instance) (responses []*ec2.SimpleResp, err error) {
	instanceIds := getInstanceIds(instances)

	ec2Connection, err := ec2Connect()
	if err != nil {
		return nil, err
	}

	for _, instance := range instances {
		nameTag := ec2.Tag{"Name", instance.InstanceId}
		gopherTag := ec2.Tag{"gopher", "true"}
		tags := []ec2.Tag{nameTag, gopherTag}

		response, err := ec2Connection.CreateTags(instanceIds, tags)
		if err != nil {
			return nil, err
		}

		responses = append(responses, response)
	}

	return responses, nil
}

func findInstances(statusCode int) (instances []ec2.Instance, err error) {
	filter := ec2.NewFilter()
	filter.Add("tag:gopher", "true")

	if statusCode != -1 {
		filter.Add("instance-state-code", strconv.Itoa(statusCode))
	}

	ec2Connection, err := ec2Connect()
	resp, err := ec2Connection.Instances(nil, filter)
	if err != nil {
		return nil, err
	}

	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			instances = append(instances, instance)
		}
	}

	return instances, nil
}

func sshClient(host string) (*ssh.ClientConn, error) {
	var auths []ssh.ClientAuth

	keypath := getKeyPath()
	k := &keyring{}
	err := k.loadPEM(keypath)
	if err != nil {
		return nil, err
	}

	auths = append(auths, ssh.ClientAuthKeyring(k))

	config := &ssh.ClientConfig{
		User: "ubuntu",
		Auth: auths,
	}

	client, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func getKeyPath() string {
	user, _ := user.Current()
	homeDirectory := user.HomeDir

	return filepath.Join(homeDirectory, ".ssh", fmt.Sprintf("%s.pem", instanceKey))
}

func handleError(err error) {
	if err != nil {
		panic(err)
	}
}
