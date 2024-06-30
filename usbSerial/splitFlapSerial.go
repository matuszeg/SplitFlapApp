package usbSerial

import (
	"errors"
	"google.golang.org/protobuf/proto"
	"math/rand"
	"sync"
	"time"
)

const (
	ForceMovementNone         ForceMovement = "none"
	ForceMovementOnlyNonBlank ForceMovement = "only_non_blank"
	ForceMovementAll          ForceMovement = "all"
	RetryTime                               = time.Millisecond * 500
)

type ForceMovement string

type EnqueuedMessage struct {
	nonce uint32
	bytes []byte // bytes with CRC32 + null ending
}

type SplitFlap struct {
	serial          SerialConnection
	outQueue        chan EnqueuedMessage
	ackQueue        chan uint32
	nextNonce       uint32
	run             bool
	lock            sync.Mutex
	messageHandlers map[gen.SplitFlapType]func(interface{})
	currentConfig   *gen.SplitFlapConfig
	numModules      int
	alphabet        []rune
}

func NewSplitFlap(serialInstance SerialConnection) *SplitFlap {
	alphabet := []rune{}
	for _, v := range cfg.SplitFlap.AlphabetESP32Order {
		alphabet = append(alphabet, v)
	}

	s := &SplitFlap{
		serial:          serialInstance,
		outQueue:        make(chan EnqueuedMessage, 100),
		ackQueue:        make(chan uint32, 100),
		nextNonce:       uint32(rand.Intn(256)),
		run:             true,
		messageHandlers: make(map[gen.SplitFlapType]func(interface{})),
		currentConfig:   nil,
		alphabet:        alphabet,
	}

	// TODO: Remove later
	s.initializeModuleList(cfg.SplitFlap.ModuleCount)

	return s
}

func (sf *SplitFlap) initializeModuleList(moduleCount int) {
	sf.numModules = moduleCount
	sf.currentConfig = &gen.SplitFlapConfig{
		Modules: []*gen.SplitFlapConfig_ModuleConfig{},
	}
	for i := 0; i < moduleCount; i++ {
		newModule := gen.SplitFlapConfig_ModuleConfig{
			TargetFlapIndex: 0,
			MovementNonce:   0,
			ResetNonce:      0,
		}

		sf.currentConfig.Modules = append(sf.currentConfig.Modules, &newModule)
	}
}

func (sf *SplitFlap) readLoop() {
	logger.Info().Msg("Read loop started")
	buffer := []byte{}
	for {
		if !sf.run {
			return
		}

		newBytes, err := sf.serial.Read()
		if err != nil {
			logger.Info().Msgf("Error reading from serial: %v\n", err)
			return
		}

		if len(newBytes) == 0 {
			continue
		}

		buffer = append(buffer, newBytes...)
		lastByte := buffer[len(buffer)-1]
		if lastByte != 0 {
			continue
		}

		sf.processFrame(buffer[:len(buffer)-1])
		buffer = []byte{}
	}
}

func (sf *SplitFlap) processFrame(decoded []byte) {
	payload, validCrc := utils.ParseCRC32EncodedPayload(decoded)
	if !validCrc {
		return
	}

	message := &gen.FromSplitFlap{}

	if err := proto.Unmarshal(payload, message); err != nil {
		logger.Info().Msgf("Failed to unmarshal message: %v\n", err)
		return
	}
	message.PrintSplitFlapState()

	switch message.GetPayload().(type) {
	case *gen.FromSplitFlap_Ack:
		nonce := message.GetAck().GetNonce()
		sf.ackQueue <- nonce
	case *gen.FromSplitFlap_SplitFlapState:
		numModulesReported := len(message.GetSplitFlapState().GetModules())

		if sf.numModules == 0 {
			sf.initializeModuleList(numModulesReported)
		} else if sf.numModules != numModulesReported {
			logger.Info().Msgf("Number of reported modules changed (was %d, now %d)\n", sf.numModules, numModulesReported)
		}
	}
}

func (sf *SplitFlap) waitingForOutgoingMessage() bool {
	return len(sf.outQueue) == 0
}

func (sf *SplitFlap) waitingForIncomingMessage() bool {
	return len(sf.ackQueue) == 0
}

func (sf *SplitFlap) writeLoop() {
	logger.Info().Msg("Write loop started")

	for {
		if !sf.run {
			logger.Info().Msg("Stop running, exiting write loop")
			return
		}

		if sf.waitingForOutgoingMessage() {
			continue
		}

		enqueuedMessage := <-sf.outQueue

		nextRetry := time.Now()
		writeCount := 0
		for {
			if !sf.run {
				logger.Info().Msg("Stop running, exiting write loop")
				return
			}

			if time.Now().After(nextRetry) {
				if writeCount > 0 {
					logger.Info().Msg("Failed to write message, resetting queue")
					sf.outQueue = make(chan EnqueuedMessage, 100)
					break
				}

				writeCount++
				sf.serial.Write(enqueuedMessage.bytes)
				nextRetry = time.Now().Add(RetryTime)
			}

			if sf.waitingForIncomingMessage() {
				continue
			}

			latestAckNonce := <-sf.ackQueue
			if enqueuedMessage.nonce == latestAckNonce {
				break
			}
		}
	}
}

func (sf *SplitFlap) SetText(text string) error {
	return sf.setTextWithMovement(text, ForceMovementNone)
}

func (sf *SplitFlap) setTextWithMovement(text string, forceMovement ForceMovement) error {
	// Transform text to a list of flap indexes (and pad with blanks so that all modules get updated even if text is shorter)
	var positions []uint32
	for _, c := range text {
		idx := sf.alphabetIndex(c)
		positions = append(positions, idx)
	}

	// Pad with blanks if text is shorter than the number of modules
	for i := len(text); i < sf.numModules; i++ {
		positions = append(positions, sf.alphabetIndex(' '))
	}

	var forceMovementList []bool
	switch forceMovement {
	case ForceMovementNone:
		forceMovementList = nil
	case ForceMovementOnlyNonBlank:
		for _, c := range text {
			forceMovementList = append(forceMovementList, sf.alphabetIndex(c) != 0)
		}
		// Pad with false if text is shorter than the number of modules
		for i := len(text); i < sf.numModules; i++ {
			forceMovementList = append(forceMovementList, false)
		}
	case ForceMovementAll:
		forceMovementList = make([]bool, sf.numModules)
		for i := range forceMovementList {
			forceMovementList[i] = true
		}
	default:
		panic("Bad movement value")
	}

	return sf.setPositions(positions, forceMovementList)
}

func (sf *SplitFlap) setPositions(positions []uint32, forceMovementList []bool) error {
	sf.lock.Lock()
	defer sf.lock.Unlock()

	if sf.numModules == 0 {
		return errors.New("cannot set positions before the number of modules is known")
	}

	if len(positions) > sf.numModules {
		return errors.New("more positions specified than modules")
	}

	if forceMovementList != nil && len(positions) != len(forceMovementList) {
		return errors.New("positions and forceMovementList length must match")
	}

	for i := 0; i < len(positions); i++ {
		sf.currentConfig.Modules[i].TargetFlapIndex = positions[i]
		if forceMovementList != nil && forceMovementList[i] {
			sf.currentConfig.Modules[i].MovementNonce = (sf.currentConfig.Modules[i].MovementNonce + 1) % 256
		}
	}

	message := &gen.ToSplitFlap{
		Payload: &gen.ToSplitFlap_SplitFlapConfig{
			SplitFlapConfig: sf.currentConfig,
		},
	}

	sf.enqueueMessage(message)
	return nil
}

func (sf *SplitFlap) enqueueMessage(message *gen.ToSplitFlap) {
	message.Nonce = sf.nextNonce
	sf.nextNonce++

	payload, err := proto.Marshal(message)
	if err != nil {
		logger.Error().Msgf("Error marshaling message: %v\n", err)
		return
	}

	newMessage := EnqueuedMessage{
		nonce: message.Nonce,
		bytes: utils.CreatePayloadWithCRC32Checksum(payload),
	}

	sf.outQueue <- newMessage

	approxQLength := len(sf.outQueue)
	// TODO: handle error in some way
	// logger.Info().Msgf("Out q length: %d\n", approxQLength)
	if approxQLength > 10 {
		logger.Info().Msgf("Output queue length is high! (%d) Is the splitflap still connected and functional?\n", approxQLength)
	}
}

func (sf *SplitFlap) requestState() {
	message := gen.ToSplitFlap{}
	message.Payload = &gen.ToSplitFlap_RequestState{
		RequestState: &gen.RequestState{},
	}

	sf.enqueueMessage(&message)
}

func (sf *SplitFlap) alphabetIndex(c rune) uint32 {
	for i, char := range sf.alphabet {
		if char == c {
			return uint32(i)
		}
	}

	return 0 // Default to 0 if character not found in alphabet
}

func (sf *SplitFlap) Start() {
	go sf.readLoop()
	go sf.writeLoop()
	sf.requestState()
}

func (sf *SplitFlap) Shutdown() {
	logger.Info().Msg("Shutting down...")
	sf.run = false
	sf.serial.Close()
	close(sf.outQueue)
	close(sf.ackQueue)
}
