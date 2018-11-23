/*
 * This is an example of a Go program to get messages from an IBM MQ
 * queue.
 *
 * The queue and queue manager name can be given as parameters on the
 * command line. Defaults are coded in the program.
 *
 * The program loops until no more messages are on the queue, waiting for
 * at most 3 seconds for new messages to arrive.
 *
 * Each MQI call prints its success or failure.
 *
 * A MsgId can be provided as a final optional parameter to this command
 * in which case we try to retrieve just a single message that matches.
 *
 */
package main

/*
  Copyright (c) IBM Corporation 2018

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the license.

   Contributors:
     Mark Taylor - Initial Contribution
*/

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/ibm-messaging/mq-golang/ibmmq"
)

var qMgrObject ibmmq.MQObject
var qObject ibmmq.MQObject

func main() {
	var msgId string

	// The default queue manager and queue to be used. These can be overridden on command line.
	qMgrName := "QM1"
	qName := "DEV.QUEUE.1"

	qMgrConnected := false
	qOpened := false

	fmt.Println("Sample AMQSGET.GO start")

	// Get the queue and queue manager names from command line for overriding
	// the defaults. Parameters are not required.
	if len(os.Args) >= 2 {
		qName = os.Args[1]
	}

	if len(os.Args) >= 3 {
		qMgrName = os.Args[2]
	}

	// Can also provide a msgid as a further command parameter
	if len(os.Args) >= 4 {
		msgId = os.Args[3]
	}

	// This is where we connect to the queue manager. It is assumed
	// that the queue manager is either local, or you have set the
	// client connection information externally eg via a CCDT or the
	// MQSERVER environment variable
	qMgrObject, err := ibmmq.Conn(qMgrName)
	if err != nil {
		fmt.Println(err)
	} else {
		qMgrConnected = true
		fmt.Printf("Connected to queue manager %s\n", qMgrName)
	}

	// Open of the queue
	if err == nil {
		// Create the Object Descriptor that allows us to give the queue name
		mqod := ibmmq.NewMQOD()

		// We have to say how we are going to use this queue. In this case, to GET
		// messages. That is done in the openOptions parameter.
		openOptions := ibmmq.MQOO_INPUT_EXCLUSIVE + ibmmq.MQOO_FAIL_IF_QUIESCING

		// Opening a QUEUE (rather than a Topic or other object type) and give the name
		mqod.ObjectType = ibmmq.MQOT_Q
		mqod.ObjectName = qName

		qObject, err = qMgrObject.Open(mqod, openOptions)
		if err != nil {
			fmt.Println(err)
		} else {
			qOpened = true
			fmt.Println("Opened queue", qObject.Name)
		}
	}

	msgAvail := true
	for msgAvail == true && err == nil {
		var datalen int

		// The GET requires control structures, the Message Descriptor (MQMD)
		// and Get Options (MQGMO). Create those with default values.
		getmqmd := ibmmq.NewMQMD()
		gmo := ibmmq.NewMQGMO()

		// The default options are OK, but it's always
		// a good idea to be explicit about transactional boundaries as
		// not all platforms behave the same way. It's also good practice to
		// set the FAIL_IF_QUIESCING flag on all verbs.
		gmo.Options = ibmmq.MQGMO_NO_SYNCPOINT | ibmmq.MQGMO_FAIL_IF_QUIESCING

		// Set options to wait for a maximum of 3 seconds for any new message to arrive
		gmo.Options |= ibmmq.MQGMO_WAIT
		gmo.WaitInterval = 3 * 1000 // The WaitInterval is in milliseconds

		// If there is a MsgId on the command line decode it into bytes and
		// set the options for matching it during the Get processing
		if msgId != "" {
			fmt.Println("Setting Match Option for MsgId")
			gmo.MatchOptions = ibmmq.MQMO_MATCH_MSG_ID
			getmqmd.MsgId, _ = hex.DecodeString(msgId)
			// Will only try to get a single message with the MsgId as there should
			// never be more than one. So set the flag to not retry after the first attempt.
			msgAvail = false
		}

		// Create a buffer for the message data. This one is large enough
		// for the messages put by the amqsput sample.
		buffer := make([]byte, 1024)

		// Now we can try to get the message
		datalen, err = qObject.Get(getmqmd, gmo, buffer)

		if err != nil {
			msgAvail = false
			fmt.Println(err)
			mqret := err.(*ibmmq.MQReturn)
			if mqret.MQRC == ibmmq.MQRC_NO_MSG_AVAILABLE {
				// If there's no message available, then I won't treat that as a real error as
				// it's an expected situation
				err = nil
			}
		} else {
			// Assume the message is a printable string, which it will be
			// if it's been created by the amqsput program
			fmt.Printf("Got message of length %d: ", datalen)
			fmt.Println(strings.TrimSpace(string(buffer[:datalen])))
		}
	}

	// The usual tidy up at the end of a program is for queues to be closed,
	// queue manager connections to be disconnected etc.
	// In a larger Go program, we might move this to a defer() section to ensure
	// it gets done regardless of other flows through the program.

	// Close the queue if it was opened
	if qOpened {
		err = qObject.Close(0)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Closed queue")
		}
	}

	// Disconnect from the queue manager
	if qMgrConnected {
		err = qMgrObject.Disc()
		fmt.Printf("Disconnected from queue manager %s\n", qMgrName)
	}

	// Exit with any return code extracted from the failing MQI call.
	if err == nil {
		os.Exit(0)
	} else {
		mqret := err.(*ibmmq.MQReturn)
		os.Exit((int)(mqret.MQCC))
	}
}
