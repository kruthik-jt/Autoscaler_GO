package main

//Importing important packages.
import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	//Package v1 contains API types that are common to all versions.
	//metav1 is a name that has been set to use instead of using the full package name (an import that has a name called aliased import).
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//Package contains the clientset to access Kubernetes API.
	"k8s.io/client-go/kubernetes"
	//Package clientcmd provides one stop shopping for building a working client from a fixed config, from a .kubeconfig file, from command line flags, or from any merged combination.
	"k8s.io/client-go/tools/clientcmd"
	//Package has the automatically generated clientset.
	//again metricsclientset is the name that we will used instead of the full package name.
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

//Struct is fields collection for grouping data together. It is used to store all the required measurements that we aquire from all pods through kubectl.
type measurementPod struct {
	PodName           string
	CPU               float64 //CPU Utilization in cores
	WindowSize        float64
	TimeStamp         int64
	CreationTimestamp int64
	isAccountedFor    bool
}

//Global Variables
var minReplica int32 = 1        //MinReplica is the minimum number of running replica in the system. we should have at least one running replica.
var maxReplica int32 = 10       //MaxReplica is the maximum number of replicas.
var desiredMetrics float64 = 50 //DesiredMetrics is average CPU Utilization in precentage. The feedback algorithm should scale up and down to maintain our target.
var requestValue float64 = 200  //Request Value is the minimum amount of resources that containers need.
var baseMilicore float64 = 1000.0

//currentMetrics function calculates current metrics by taking:
//Inputs: CPU Utlization in Nanocore, Number of Running Pods, Average Utilization Array
//Output: Average Utlization Array for Every 1 min
func currentMetrics(m [][10]measurementPod, measurementIndex int32, pMeasurementindex int32, totalUtilizationArray []float64, totalNumPodsArray []int32, index int32) {
	var totalCpu float64 = 0
	//var currMetrics float64 = 0
	var numPods int32 = 0

	//Verify if the last measurement is accounted for in scaling algorithm.
	// If not, take that into account as well.
	//The loop start with the last measuremt is accounted and ends at current measurements,
	//so it will take six measurements for 1 min.
	//prvious measurement index initaly is 0
	//measurement index is the current measurement.
	for j := pMeasurementindex; j < measurementIndex+1; j++ {
		//Another loop to check inside each measurements for the number of running containers
		for i := 0; i < int(maxReplica); i++ {
			//before we calculate the total CPU and number of containers.
			//Check the Pod Name if is not dummy (this means we have a CPU value).
			//Also check if the measurement is accounted (false).
			//True --> we take it into account.
			//False --> we didn't take, so we have to consider it.
			mustTakeMeasurementIntoAccount := m[j][i].PodName != "dummy" && !m[j][i].isAccountedFor
			if mustTakeMeasurementIntoAccount {
				//Calculating the total CPU in cores and containers number.
				//Then, we set the measurement to True to not retake into account.
				totalCpu += m[j][i].CPU
				m[j][i].isAccountedFor = true
				numPods = numPods + 1
			}
		}
	}
	//fmt.Println("Total CPU  [CurrentMetrics Function] ", totalCpu)
	//fmt.Println(" Total number of pods [CurrentMetrucs Function] ", numPods)
	//Calculate the average utlization every 1 min
	totalUtilizationArray[index] = totalCpu
	//Store the average utlization in an array to be used to calculate the current metrics in precentage.
	totalNumPodsArray[index] = numPods
	//fmt.Println("totalUtilizationArray [CurrentMetrics Function] ", totalUtilizationArray[index], "totalCpu", totalCpu)
	//fmt.Println("totalNumPodsArray [CurrentMetrucs Function] ", totalNumPodsArray[index], "numPods", numPods)
}

//scalingAlgorithm function calculates the Desired Replicas based on Algorithm 1.
//Input : Current Metrics, Desired Metrics, Current Relicas.
//Output: Desired Replicas.
func scalingAlgorithm(replicaCount int32, CurrentMetricsPrecentage float64) int32 {
	var desiredReplicas int32 = 0

	//HPA equation: Ceiling (Current Replicas * Current Metrics / Desired Metrics).
	desiredReplicas = int32(math.Ceil(float64(replicaCount) * CurrentMetricsPrecentage / desiredMetrics))
	//Check policy (minReplica and maxReplica)
	//After we calculate the desired replicas, we should check the autoscaling policy.
	//also we should store the prvious decision whether is up or down.
	//recall decision check function
	desiredReplicas = getDesiredNumReplica(desiredReplicas, replicaCount)
	fmt.Println("Desired Replica = ", desiredReplicas, " Current Metrics = ", CurrentMetricsPrecentage, " Current Replica = ", replicaCount)
	return desiredReplicas
}

//Check policy and get the previous decsion
//After we calculate the desired replicas, we should check the autoscaling policy.
//Desired Replica cannot be greater than the Max Replica.
//Desired Replica cannot be less than the Min Replica.
func getDesiredNumReplica(desiredReplica int32, replicaCount int32) int32 {
	if desiredReplica > maxReplica {
		desiredReplica = maxReplica
	} else if desiredReplica < minReplica {
		desiredReplica = minReplica
	}
	return desiredReplica
}

//This function is called to check for the difference between desired and current replica count
//The current replica should be updated in the next time interval.
//The number of containers should be installed and running.
//if the current replica is not updated, we should wait until the deployment updated by scale up or down based on the desired replica.
func PollReplicas(desiredReplicaCount int32, currReplicaCount int32, kube_cs *kubernetes.Clientset) {
	fmt.Println("THIS WILL SHIFT THE TIME LINE...")
	for {
		fmt.Println("Current Replica = ", currReplicaCount, "Desired Replica", desiredReplicaCount)
		time.Sleep(1 * time.Second)
		//Updating the current replica in the deployment.
		phpDeployment, err := kube_cs.AppsV1().Deployments("default").GetScale(context.TODO(), "php-apache", metav1.GetOptions{})
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		//current replica count is updated by calling the phpdeployment and checked for the synchronacy
		currReplicaCount = phpDeployment.Status.Replicas
		if desiredReplicaCount == currReplicaCount {
			fmt.Println("System in Sync")
			return
		}
	}
}

//This function is used to indicate if there is any duplicate value
//Input: measurement index, current pod name, current metrics time stamp, previous measurement.
//Output: the container number in the current measurement that has the same pod name and same timestamp.
func FindPreviousIndexDuplicateContainer(m [][10]measurementPod, index int32, name string, timestamp int64) int32 {
	//check each container in a measurement
	for i := 0; i < int(maxReplica); i++ {
		//if the pod name is equal to prvious name AND if the timestamp is equal to the previous measurement
		//return the number of container for the next measurement int32(i).
		mustCheckDuplicateMeasurement := name == m[index-1][i].PodName && int64(m[index-1][i].TimeStamp) == timestamp
		if mustCheckDuplicateMeasurement {
			return int32(i)
		}
	}
	//if the values does not match with the previous measurement, return -1
	return -1
}

//This function is to get the pods' information form kubelet through the metrics server
func updateMetricsInArray(m [][10]measurementPod, measurementIndex int32, containerIndex int32,
	podMetric v1beta1.PodMetrics, container v1beta1.ContainerMetrics, isAccountedFor bool) {
	//These value we don't have any control
	//metricTimestamp is timestamp of kubelet (we consider metrics timestamp to be 50 sec.)
	//CreationTimestamp is the polling time (here is set to be 10 second)
	m[measurementIndex][containerIndex].PodName = podMetric.ObjectMeta.Name
	m[measurementIndex][containerIndex].WindowSize = podMetric.Window.Duration.Seconds()
	m[measurementIndex][containerIndex].TimeStamp = podMetric.Timestamp.Time.Unix()
	m[measurementIndex][containerIndex].CreationTimestamp = podMetric.CreationTimestamp.Time.Unix()
	m[measurementIndex][containerIndex].CPU = container.Usage.Cpu().ToDec().AsApproximateFloat64()
	//We are controlling this value by checking if the provided measurment has been taken into account or not
	//True: we already take it into account
	//False: we didn't take it yet. (considering it is not accounted for calculation of CPU)
	m[measurementIndex][containerIndex].isAccountedFor = isAccountedFor
}

//This function is to calculate under-provisioning accuracy that allocates the number of missing resources that are required.
//AND to calculate over-provisioning accuracy that defines the number of unused resources during the time interval.
//INPUT: Desired Replica, Current Replica, Previous Under Provisioning Accuracy, and Previous Over Provisioning Accuracy
//OUTPUT: Current Under Provisioning Accuracy, and Current Over Provisioning Accuracy
func ProvisioningAccuracy(desiredReplicaCount int32, currReplicaCount int32, underProvAccuracy float64, overProvAccuracy float64) (float64, float64) {
	var deltaTime int32 = 5 // 1 for 1 min and 5 for 5 min

	//Calculate max(desiredReplica-CurrentReplica,0) and then divide by desiredReplica and multiply by delta time.
	//This is inside the sum
	totalUnderAccuracy := math.Max((float64(desiredReplicaCount)-float64(currReplicaCount)), 0) / float64(desiredReplicaCount) * float64(deltaTime)
	//Calculate the total under-provisioning accuracy from time = 1 until current time
	underProvAccuracy = underProvAccuracy + totalUnderAccuracy

	//Calculate max(CurrentReplica-DesiredReplica,0) and then divide by desiredReplica and multiply by delta time.
	//This is inside the sum
	totalOverAccuracy := math.Max((float64(currReplicaCount)-float64(desiredReplicaCount)), 0) / float64(desiredReplicaCount) * float64(deltaTime)
	//Calculate the total over-provisioning accuracy from time = 1 until current time
	overProvAccuracy = overProvAccuracy + totalOverAccuracy

	//Return the under- and over-provisioing accuracy
	return underProvAccuracy, overProvAccuracy
}

//This is Signum function as GOLang doesn't have this math function inside its package.
//This function will be used in the ProvisioningTimeshare function
//INPUT: x  (float)
//OUTPUT: if x is any positive number return +1, if else x is any negative number return -1. Otherwise return 0
func Sgn(a float64) int {
	switch {
	case a < 0:
		return -1
	case a > 0:
		return +1
	}
	return 0
}

//measure the prediction duration when the autoscale system is under-provisioned and over-provisioned, respectively, during the experiment time
//INPUT: Desired Replica, Current Replica, Previous Under Provisioning Time, and Previous Over Provisioning Time
//OUTPUT: Current Under Provisioning Time, and Current Over Provisioning Time
func ProvisioningTimeshare(desiredReplicaCount int32, currReplicaCount int32, underProvTime float64, overProvTime float64) (float64, float64) {
	var deltaTime int32 = 5 // 1 for 1 min and 5 for 5 min

	//Recall signum function to calculate the different between desire replica and current replica
	//UNDER: DesiredReplica - CurrentReplica
	signumUnder := Sgn(float64(desiredReplicaCount) - float64(currReplicaCount))
	//OVER: CurrentReplica - DesiredReplica
	signumOver := Sgn(float64(currReplicaCount) - float64(desiredReplicaCount))

	//Calculate max(desiredReplica-CurrentReplica,0) and then multiply by delta time.
	//This is inside the sum
	resultUnderTime := math.Max(float64(signumUnder), 0) * float64(deltaTime)
	//Calculate the total under-provisioning Time from time = 1 until current time
	underProvTime = underProvTime + resultUnderTime

	//Calculate max(CurrentReplica-DesiredReplica,0) and then multiply by delta time.
	//This is inside the sum
	resultOverTime := math.Max(float64(signumOver), 0) * float64(deltaTime)
	//Calculate the total over-provisioning time from time = 1 until current time
	overProvTime = overProvTime + resultOverTime

	//Return the under- and over-provisioing time
	return underProvTime, overProvTime
}

func main() {
	arg := os.Args[1:]
	//command line input
	//when you run the code, we can indicate the min replica based on the current status
	//if we initally have 3 pods --> ./podmetrics 3
	//3 will be the min replica
	minReplicaArg, err := strconv.Atoi(arg[0])

	//Initalize variables.
	minReplica = int32(minReplicaArg) //passing command line input to minReplica
	var measurements [2000][10]measurementPod
	var replicaCount int32 = minReplica //replicaCount is current min replica
	var measurementIndex int32 = 0      //Meaurement Index stats with 0 until 2000
	var containerIndex int32 = 0        //starting with container index = 0 (0--> 9 "maxReplica")
	var pMeasurementindex int32 = 0     //Represents the number of last accounted measurement
	var iter int32 = 0
	var totalUtilizationArray [2000]float64 //Array to store average utilization that we will calaculate every 1 min
	var totalNumPodsArray [2000]int32
	var index int32 = 0            // Utilization Index start with 0 until 2000
	var scalingIteration int32 = 0 //Iteration to do the scaling --> here every 5 mins
	var fiveAverageMeaurements int32 = 0
	var TotalAverageUtlization float64 = 0
	var numMeasurementCount int32 = 5 //Number of measurement we take into account for scaling
	var AverageUtlizationMeasurements float64 = 0
	var totalExperimentDuration int = 5
	var underProvAccuracy float64 = 0
	var overProvAccuracy float64 = 0
	var underProvTime float64 = 0
	var overProvTime float64 = 0

	//Loading kubernetes configuration from a specific location.
	//Here e.g. the config file in this path "C:/Users/HP/.kube/config"
	//Change the file path based on where you store it.
	//building a working client from a kubeconfig file
	kubeconfig := flag.String("kubeconfig", "C:/Users/HP/.kube/config", "location of kubeconfig")
	//This line is to get kuberenetes config.
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	//creating a new Clientset for the kubeconfig file
	metric_cs, err := metricsclientset.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	//creating a new Clientset for the kubeconfig file
	kube_cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	for {
		//intitiating podMetrics array to get and store podmetrics data
		podMetrics, err := metric_cs.MetricsV1beta1().PodMetricses("default").List(context.Background(), (metav1.ListOptions{LabelSelector: "run=php-apache"}))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		//Iterate over the results found.
		//Initalize the number of container onec we start a new measurment index (here intialize every 10 second)
		containerIndex = 0

		//Ranging over the each podmetrics container information(podname, windowsize, Creationtimestamp,cpu) and storing it in measurement array
		for _, podMetric := range podMetrics.Items {
			//Iterate over container --> you need this line in case you have more than one container.
			podContainers := podMetric.Containers
			//number of live containers should equal to length of items in pod metrics
			var numLiveContainers int32 = int32(len(podMetrics.Items))

			//Creating struct specifying field names (PodName, WindowSize, TimeStamp, CreationTimestamp, and CPU).
			//Storing struct field names in measurements array. We only store the live or running containers/pods.
			for _, container := range podContainers {
				//Considering the first measurement as fresh measurement
				if measurementIndex == 0 {
					//Recalling updateMetricsInArray function to get all the pods' information and if the measurement was accounted
					updateMetricsInArray(measurements[:][:], measurementIndex, containerIndex, podMetric, container, false)
					//If it is not the first measument check the following
				} else {
					//Recall a function to find privous index duplicate container.
					//for every running container, we check the pod name and time stamp by comparing with the previous measurement.
					prevDupIndex := FindPreviousIndexDuplicateContainer(measurements[:][:], measurementIndex, podMetric.ObjectMeta.Name, podMetric.Timestamp.Time.Unix())
					//if the previous duplicate index was grater than -1
					//this measuremtn is duplicate
					duplicateMeasurementFound := prevDupIndex > -1
					if duplicateMeasurementFound {
						//if we have a duplicate value
						//If the current measurement is duplicate, the previous measurements are copied into the current measurement and previous measurement is set to true
						//this is helpful for the next measurements comparision
						measurements[measurementIndex][containerIndex] = measurements[measurementIndex-1][prevDupIndex]
						measurements[measurementIndex-1][prevDupIndex].isAccountedFor = true
					} else {
						//if the measurement is a fresh measurement, poll all the pod information.
						updateMetricsInArray(measurements[:][:], measurementIndex, containerIndex, podMetric, container, false)
					}
				}
				//Incrementing container index for the next container
				containerIndex += 1
			}
			//set dummy value for the unused containers
			//dummy containers will be excluded while calculating current metrics
			for i := numLiveContainers; i < maxReplica; i++ {
				measurements[measurementIndex][i].PodName = "dummy"
			}
		}
		//Incrementing the iteration number every 10 second
		iter += 1

		//Podmetrics is pulled for every 10 sec, iter is increased
		// For every 1 min, the scaling algorithm is called (iter =6)
		if iter == 6 {
			//checking and updating the current number of replicas
			phpDeployment, err := kube_cs.AppsV1().Deployments("default").GetScale(context.TODO(), "php-apache", metav1.GetOptions{})
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			//check if the  current replica equalt to the desired replica
			if replicaCount != phpDeployment.Status.Replicas {
				currentReplicaCount := phpDeployment.Status.Replicas
				PollReplicas(replicaCount, currentReplicaCount, kube_cs)
			}

			//current replica to be used in scaling algorithm
			replicaCount = phpDeployment.Status.Replicas

			//fmt.Println("Scaling Iteration [Main Function] ", scalingIteration,"Repica Count ", replicaCount)

			//if Scaling Iteration = 4, this means that we have collect value for 5 min and it's time to calculate the desired Replica.
			if scalingIteration == 4 {

				//Calling the current metrics function to calulcate the average of this time
				currentMetrics(measurements[:][:], measurementIndex, pMeasurementindex, totalUtilizationArray[:], totalNumPodsArray[:], index)

				//Check the average Utilization array if it has at lease 5 values to calclate the currenet metrics in precentage.
				for i := fiveAverageMeaurements; i < fiveAverageMeaurements+5; i++ {
					//Calculate the total Average Utlization of last 5 measurements in the array
					AverageUtlizationMeasurements = totalUtilizationArray[i] / float64(totalNumPodsArray[i])
					TotalAverageUtlization = TotalAverageUtlization + AverageUtlizationMeasurements
					//fmt.Println("totalUtilizationArray ", totalUtilizationArray[i], "totalNumPodsArray ", totalNumPodsArray[i], "TotalAverageUtlization ", TotalAverageUtlization)
				}
				//Calculate the Current Metrics in Precentage based on Algorithm 2 explaination
				// Current Metrics = total Average Utlization of the last 5 minutes * 1000 * 100 / (request value  * number of measurment "Here is 5")
				var CurrentMetricsPrecentage float64 = TotalAverageUtlization * baseMilicore * 100.0 / (requestValue * float64(numMeasurementCount))
				//Recall the scalingAlgorithm function to calculate the desired replica and updating the previous decision
				//passing replica count, the measurement array to calculate current metrics
				//here the desired replica is the current replica
				replicaCount = scalingAlgorithm(replicaCount, CurrentMetricsPrecentage)
				//once we measured the desired replica, set the prvious measurment index to current measuremnt index
				//in this case, everytime we will check from the last accounted measurment to the current one.
				pMeasurementindex = measurementIndex

				//Set current replica to use it as an input in the four evaluation metrics
				currentReplicaCount := phpDeployment.Status.Replicas

				//recall Provisioing function to return the accuracy and timeshare for under- and over-provisioning
				underProvAccuracy, overProvAccuracy = ProvisioningAccuracy(replicaCount, currentReplicaCount, underProvAccuracy, overProvAccuracy)
				underProvTime, overProvTime = ProvisioningTimeshare(replicaCount, currentReplicaCount, underProvTime, overProvTime)

				//Calculate under- and over-provisioning accuracy in presentage.
				underProvAccuracyPercentage := underProvAccuracy * 100 / float64(totalExperimentDuration)
				overProvAccuracyPercentage := overProvAccuracy * 100 / float64(totalExperimentDuration)

				//Calculate under- and over-provisioning timeshare in presentage.
				underProvTimePercentage := underProvTime * 100 / float64(totalExperimentDuration)
				overProvTimePercentage := overProvTime * 100 / float64(totalExperimentDuration)

				//Print all the values to do the evaluation graph.
				fmt.Println("UnderProvisioingAccuracy ", underProvAccuracyPercentage,
					"OverProvisioingAccuracy ", overProvAccuracyPercentage,
					"UnderProvisioingTime", underProvTimePercentage,
					"OverProvisioingTime", overProvTimePercentage)
				//Setting the number of replicas for the next following time
				phpDeployment.Spec.Replicas = replicaCount

				//Update the current replica in the deployment
				kube_cs.AppsV1().Deployments("default").UpdateScale(context.TODO(), "php-apache", phpDeployment, metav1.UpdateOptions{})

				//reintialize the iteration to 0, to call caluclate the average utlization 1 min
				iter = 0

				//reintialize the scaling iteration to 0, to call scaling algorithm after 5 min
				scalingIteration = 0

				//Increment the Average Utilization array index by 1 every 1 min
				index += 1

				//Increment by five to consider new five measurements in the next 5 min
				fiveAverageMeaurements = fiveAverageMeaurements + 5

				//Reintialize the total Average so every time we enter the for loop, we don't consider the previous measuerements.
				TotalAverageUtlization = 0

				//Increament the duration time by 1 if the scaling was every 1 min
				totalExperimentDuration += 5 //5

				//after printing all the measurement (current replica, desired replica, and current metrics)
				//sleep for 10 second.
				time.Sleep(10 * time.Second)
			} else {
				currentMetrics(measurements[:][:], measurementIndex, pMeasurementindex, totalUtilizationArray[:], totalNumPodsArray[:], index)
				//once we measured the desired replica, set the prvious measurment index to current measuremnt index
				//in this case, everytime we will check from the last accounted measurment to the current one.
				pMeasurementindex = measurementIndex

				//reintialize the iteration to 0, to call scaling algorithm after 1 min
				iter = 0

				//Increment the scaling iteration by 1 as it is not time to call the scaling algorithm.
				scalingIteration += 1

				//Increment the Average Utilization array index by 1 every 1 min
				index += 1

				//after printing all the measurement (current replica, desired replica, and current metrics)
				//sleep for 10 second.
				time.Sleep(10 * time.Second)
			}

		} else {
			//If the iteration is not equal to 6 (we still need to collect measurement before we scale)
			//sleep for 10 second and recollect pods' information
			time.Sleep(10 * time.Second)
		}
		//increament the measuremnt index.
		measurementIndex = measurementIndex + 1
	}

}
