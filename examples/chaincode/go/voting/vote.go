/**
* This is a very simple voting app that needs to be improved. WIP
* Args are the vote candidates
*/
package main

import (
        "errors"
        "fmt"
        "strconv"
		"bytes"
		"github.com/hyperledger/fabric/accesscontrol/impl"
        "github.com/hyperledger/fabric/core/chaincode/shim"
)

type VoteChaincode struct {
}

//Init the chaincode asigned the value "0" to the counter in the state.
func (t *VoteChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	
	// Check that the caller has the "assigner" role
	isOk, _ := stub.VerifyAttribute("role", []byte("assigner"))
	if !isOk {
		return nil, errors.New("Wrong role. Expected: assigner")
	}
	
	// Add candidates
	// Number of candidates
	stub.PutState("counter", []byte(strconv.Itoa(len(args))))
	// Iterate over the candidates
	for i := 0; i < len(args); i++ {
		// Set the number of vote for the candidate to 0
		err := stub.PutState(strconv.Itoa(i), []byte(args[i]))
		stub.PutState(args[i], []byte("0"))
		if err != nil {
			return nil, err
		}
	}
	
	// Set the vote status to 0; 0 = VOTING. 1 = DONE
	err := stub.PutState("status", []byte("0"))
	if err != nil {
		return nil, err
	}
	
	// Get the owner (caller)
	user, err := impl.NewAccessControlShim(stub).ReadCertAttribute("userid")
	if err != nil {
		return nil, err
	}

	if len(user) == 0 {
		fmt.Printf("Invalid userid. Empty.")
		return nil, errors.New("Invalid userid. Empty.")
	}	
	// Set the vote owner
	stub.PutState("owner", user)

	return nil, nil
}


//Invoke Transaction makes increment counter
func (t *VoteChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
        if function != "vote" && function != "close" {
                return nil, errors.New("Invalid invoke function name. Expecting \"vote\"")
        }
		
		if function == "vote" {
			return t.vote(stub, args)
		} else if function == "close" {
			return t.closeVote(stub, args)
		}

        return nil, nil
}

// Vote for the given candidate
func (t *VoteChaincode) vote(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Execting 1 (candidate to vote for)")
	}
	// Get the vote status
	status, err := stub.GetState("status")
	if err != nil {
		return nil, err
	}
	
	voteStatus,err := strconv.Atoi(string(status))
	 if err != nil {
		return nil, err
	}			
	// Check that the vote is not over
	if voteStatus == 1 {
		return nil, errors.New("This vote is over")
	}
	
	// Get the user id
	user, err := impl.NewAccessControlShim(stub).ReadCertAttribute("userid")

	if err != nil {
		return nil, err
	}
	
	// Check if the user has already voted	
	hasVoted, err := stub.GetState(string(user))

	if err != nil {
		return nil, err
	}
	if(string(hasVoted) == "1") {
		// The user has already voted
		return nil, errors.New("The user has already voted")
	}
	
	// Vote for the given candidate
	numberOfVote, err := stub.GetState(args[0])
	if err != nil {
		return nil, errors.New("Candidate not found")
	}
	if len(numberOfVote) == 0 {
		return nil, errors.New("Candidate not found")
	}
	
	newNumber,err := strconv.Atoi(string(numberOfVote))
	newNumber = newNumber + 1
	if err != nil {
		return nil, err
	}
	// Increment the number of vote
	err = stub.PutState(args[0], []byte(strconv.Itoa(newNumber)))
	if err != nil {
		return nil, errors.New("Error increment the number of vote for the given candidate")
	}
	
	// The user has voted
	err = stub.PutState(string(user), []byte("1"))
	if err != nil {
		return nil, err
	}
		
	return nil, nil
}

// Close the vote
func (t *VoteChaincode) closeVote(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	// Close the vote
	if len(args) == 0 {
		// Only the owner can close the vote
		// Get the owner userid
		owner, err := stub.GetState("owner")
		if err != nil {
			return nil, err
		}
		// Get the caller userid
		user, err := impl.NewAccessControlShim(stub).ReadCertAttribute("userid")
		if err != nil {
			return nil, err
		}
		// Check if the caller is the vote owner
		if string(owner) == string(user) {
			// Close the vote
			// Set the vote status to 1; 0 = VOTING. 1 = DONE
			err = stub.PutState("status", []byte("1"))
			if err != nil {
				return nil, err
			}	
		} else {
			return nil, errors.New("Only the vote owner can close the vote")
		}
		
	} else {
		return nil, errors.New("Incorrect number of arguments. Execting 0")
	}
	return nil, nil
}

// Query callback representing the query of a chaincode
func (t *VoteChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
        if function != "getCandidates" && function != "getResults" {
                return nil, errors.New("Invalid query function name. Expecting \"getCandidates\" or \"getResults\"")
        }
        var err error
		
		if(function == "getResults") {
			
			// Get the vote status
			status, err := stub.GetState("status")
			if err != nil {
				jsonResp := "{\"Error\":\"Failed to get state for status\"}"
				return nil, errors.New(jsonResp)
			}
			
			// Check that the vote is over before sneding the results
			voteStatus,err := strconv.Atoi(string(status))
			 if err != nil {
                jsonResp := "{\"Error\":\"Failed to get state for status\"}"
                return nil, errors.New(jsonResp)
			}
			
			if voteStatus == 0 {
				jsonResp := "{\"Error\":\"The vote is not over yet. You can't watch the results\"}"
				return nil, errors.New(jsonResp)
			}		
			
		}
		
        // Get the number of candidates
        counter, err := stub.GetState("counter")
        if err != nil {
                jsonResp := "{\"Error\":\"Failed to get state for counter\"}"
                return nil, errors.New(jsonResp)
        }

        if counter == nil {
                jsonResp := "{\"Error\":\"Nil amount for counter\"}"
                return nil, errors.New(jsonResp)
        }
		
		var cInt int
		cInt, err = strconv.Atoi(string(counter))
		// Create a json response
		var mys bytes.Buffer
		mys.WriteString("{\"voteOwner\":\"")
		voteOwner, err := stub.GetState("owner")
		if err != nil {
			return nil, err
		}
		mys.WriteString(string(voteOwner))
		mys.WriteString("\",\"candidates\":[")
		for i := 0; i < cInt; i++ {
			// Get the current candidate name
			currentName, err := stub.GetState(strconv.Itoa(i))
			if err != nil {
				return nil, err
			}
			// Get the current vote count
			currentVoteCount, err := stub.GetState(string(currentName))
			if err != nil {
				return nil, err
			}
			if i > 0 {
				mys.WriteString(",")
			}
			mys.WriteString("{\"name\":\"")
			mys.WriteString(string(currentName))
			mys.WriteString("\"")
			if(function == "getResults") {
				mys.WriteString(",\"votes\":")
				mys.WriteString(string(currentVoteCount))
			}
			mys.WriteString("}")
		}	
		mys.WriteString("]}")
        jsonResp := mys.String()
        return []byte(jsonResp), nil
}

func main() {
        err := shim.Start(new(VoteChaincode))
        if err != nil {
                fmt.Printf("Error starting Simple chaincode: %s", err)
        }
}

