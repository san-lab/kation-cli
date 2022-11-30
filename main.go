// Copyright (c) 2018 Clearmatics Technologies Ltd

package main

import (
	"os"
	"fmt"
	"github.com/clearmatics/ion-cli/cli"
)

var rpcUrl = "http://localhost:8545"
var rpcUrlClique = "http://localhost:8645"
var accName = "d1e6"
var keysPath = "/home/researchSth/ion/keys.json"
var keysPass = "123"
var defaultGas = "3000000"
var defaultAmount = "0"

var cliqueContractPath = "/home/researchSth/ion/contracts/validation/Clique.sol"
var cliqueContractAddress = "0xB04511f1B29F6769380C56332B44483D0c88Bb0E"
var funCliqueRegisterChain = "RegisterChain"
var funCliqueSubmitBlock = "SubmitBlock"
var ethStorageContractAddress = "0x0E2D1eF3088d95777b3f59f3E0181a048C4E52CE"

var validators = "0x581d470ed7fa33c4b56f2785b2d7af470f6b127e"

//var ethStoreAddress = "0x0E2D1eF3088d95777b3f59f3E0181a048C4E52CE"
//var verifierAddress = "0xA421D33ddB85150419f0f6E5d55B9DB2f40957D2"

//var inputArgs = []string{}
//var inputArgs = []string{"15"}

//var constructorArgs = []string{}
//var constructorArgs = []string{ethStoreAddress, verifierAddress}


func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s [d/i/p] [parameters...]\n", os.Args[0])
                return
        }
	switch os.Args[1] {
	case "d":
		if len(os.Args) < 3 {
			fmt.Printf("Usage: %s d [contractPath] [args...]\n", os.Args[0])
			return
		}
		contractPath := os.Args[2]

		constructorArgs := []string{}
		for i := 3; i < len(os.Args); i++ {
			constructorArgs = append(constructorArgs, os.Args[i])
		}
		cli.CompileAndDeploy(rpcUrl, accName, keysPath, keysPass, contractPath, defaultGas, constructorArgs)

	case "i":
		if len(os.Args) < 5 {
                        fmt.Printf("Usage: %s i [contractPath] [contractAddress] [methodName] [args...]\n", os.Args[0])
                	return
		}
		contractPath := os.Args[2]
		contractAddress := os.Args[3]
		funName := os.Args[4]

		inputArgs := []string{}
                for i := 5; i < len(os.Args); i++ {
                        inputArgs = append(inputArgs, os.Args[i])
                }
		cli.InteractWithContract(rpcUrl, accName, keysPath, keysPass, contractPath, defaultGas, defaultAmount, contractAddress, funName, inputArgs)

	case "p":
                if len(os.Args) < 10 {
                        fmt.Printf("Usage: %s p [contractPath] [contractAddress] [methodName] [chainId] [blockHash] [previousBlockHash] [contractEmittedAddress] [transactionHash] [args...]\n", os.Args[0])
                        return
                }
		contractPath := os.Args[2]
                contractAddress := os.Args[3]
		funName := os.Args[4]

		chainId := os.Args[5]
                blockHash := os.Args[6]
		previousBlockHash := os.Args[7]
                contractEmittedAddress := os.Args[8]
                transactionHash := os.Args[9]

		// "Register" chain
		inputArgs := []string{chainId, validators, previousBlockHash, ethStorageContractAddress}
		fmt.Println(inputArgs)
		cli.InteractWithContract(rpcUrl, accName, keysPath, keysPass, cliqueContractPath, defaultGas, defaultAmount, cliqueContractAddress, funCliqueRegisterChain, inputArgs)

		// Submit block
		rlpUnsignedHeader, rlpSignedHeader := cli.GetRlpHeaders(rpcUrlClique, blockHash)
		inputArgs = []string{chainId, rlpUnsignedHeader, rlpSignedHeader, ethStorageContractAddress}
		fmt.Println(inputArgs)
		cli.InteractWithContract(rpcUrl, accName, keysPath, keysPass, cliqueContractPath, defaultGas, defaultAmount, cliqueContractAddress, funCliqueSubmitBlock, inputArgs)

		// Submit proof
                proof := cli.GetProof(rpcUrlClique, transactionHash)
		inputArgs = []string{chainId, blockHash, contractEmittedAddress, proof}
		for i := 10; i < len(os.Args); i++ {
                        inputArgs = append(inputArgs, os.Args[i])
                }
		fmt.Println(inputArgs)
		cli.InteractWithContract(rpcUrl, accName, keysPath, keysPass, contractPath, defaultGas, defaultAmount, contractAddress, funName, inputArgs)
	default:
		fmt.Println("Wrong parameters")
	}
}
