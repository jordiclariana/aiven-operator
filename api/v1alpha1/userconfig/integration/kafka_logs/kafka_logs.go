// Code generated by user config generator. DO NOT EDIT.
// +kubebuilder:object:generate=true

package kafkalogsuserconfig

type KafkaLogsUserConfig struct {
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=249
	// Topic name
	KafkaTopic string `groups:"create,update" json:"kafka_topic"`
}
