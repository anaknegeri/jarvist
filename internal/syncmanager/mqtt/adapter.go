package mqtt

type LoggerAdapter struct {
	sender *Sender
}

func NewLoggerAdapter(sender *Sender) *LoggerAdapter {
	return &LoggerAdapter{
		sender: sender,
	}
}

func (a *LoggerAdapter) Publish(topic string, payload interface{}) error {
	_, err := a.sender.SendData(topic, payload)
	return err
}

func (a *LoggerAdapter) IsConnected() bool {
	return a.sender.client.IsConnected()
}
