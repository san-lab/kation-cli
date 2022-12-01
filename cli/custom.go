// Copyright (c) 2018 Clearmatics Technologies Ltd

package cli

import (
        "context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"encoding/hex"
	"reflect"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/clearmatics/ion-cli/utils"

	"github.com/clearmatics/ion-cli/config"
	contract "github.com/clearmatics/ion-cli/contracts"
)

var defaultContractName = "myContract" // Name doesnt matter at all cause it isnt going to persist

var ctx = context.Background()

var ethClient *EthClient = nil
var accounts map[string]*config.Account = make(map[string]*config.Account)
var contracts map[string]*contract.ContractInstance = make(map[string]*contract.ContractInstance)

func CompileAndDeploy(rpcUrl string, accName string, keysPath string, keysPass string, contractPath string, defaultGas string, constructorArgs []string) error {
	err := connectToClient(rpcUrl)
	if err != nil {
                return err
        }
	err = addAccount(accName, keysPath, keysPass)
	if err != nil {
                return err
        }
	err = compileContract(defaultContractName, contractPath, contracts)
	if err != nil {
                return err
        }
	deployContract(constructorArgs, defaultContractName, accName, defaultGas)
	return nil
}

func InteractWithContract(rpcUrl string, accName string, keysPath string, keysPass string, contractPath string, defaultGas string, defaultAmount string, contractAddress string, funName string, inputArgs []string) error {
	err := connectToClient(rpcUrl)
	if err != nil {
                return err
        }
	err = addAccount(accName, keysPath, keysPass)
        if err != nil {
                return err
        }
	err = compileContract(defaultContractName, contractPath, contracts)
        if err != nil {
                return err
        }
	transactionMessage(inputArgs, defaultContractName, funName, accName, contractAddress, defaultAmount, defaultGas)
	return nil
}

func GetRlpHeaders(rpcUrl string, blockHash string) (string, string, error) {
	err := connectToClient(rpcUrl)
        if err != nil {
                return "", "" , err
        }
	block, _, err := getBlockByHash(ethClient, blockHash)
	if err != nil {
		fmt.Println("Rlp headers error")
		return "", "", err
	}
	_signedBlock, _unsignedBlock := RlpEncodeClique(block)
	//signedBlock := string(_signedBlock)
	//signedBlock = "0x" + signedBlock
        //unsignedBlock := string(_unsignedBlock)
        //unsignedBlock = "0x" + unsignedBlock
	unsignedBlock := fmt.Sprintf("0x%x", _unsignedBlock)
	signedBlock := fmt.Sprintf("0x%x", _signedBlock)
	return unsignedBlock, signedBlock, nil
}


func GetProof(rpcUrl string, transactionHash string) (string, error) {
        err := connectToClient(rpcUrl)
        if err != nil {
                return "", err
        }

	_proof := getProof(ethClient, transactionHash)
	proof := fmt.Sprintf("0x%x", _proof)
	return proof, nil
}

func connectToClient(url string) error {
	client, err := getClient(url)
	if err != nil {
		fmt.Println("Could not connect to client.\n")
		return err
	}
	ethClient = client
	return nil
}

func addAccount(name string, path string, pass string) error {
	auth, key, err := config.InitUser(path, pass)
	if err != nil {
		fmt.Println(err)
		return err
	}
	account := &config.Account{Auth: auth, Key: key}
	accounts[name] = account
	fmt.Println("Account added succesfully.")
	return nil
}

func compileContract(contractName string, pathToContract string, contracts map[string]*contract.ContractInstance) error {
	err := addContractInstance(pathToContract, contractName, contracts)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println("Added!")
	return nil
}

//TODO error checking
func deployContract(constructorArgs []string, contractName string, accountName string, customGasLimit string) {
		contractInstance := contracts[contractName]
		if contractInstance == nil {
			errStr := fmt.Sprintf("Contract instance %s not found.\nUse \taddContractInstance [name] [path/to/solidity/contract]\n", contractName)
			fmt.Println(errStr)
			return
		}

		binStr, abiStr := contract.GetContractBytecodeAndABI(contractInstance.Contract)

		account := accounts[accountName]
		if account == nil {
			errStr := fmt.Sprintf("Account %s not found.\nUse \taddAccount [name] [path/to/keystore] \n", accountName)
			fmt.Println(errStr)
			return
		}

		gasLimit, err := strconv.ParseUint(customGasLimit, 10, 64)
		if err != nil {
			fmt.Println(err)
			return
		}

		constructorInputs, err := customParseMethodParameters(constructorArgs, contractInstance.Abi, "")
		//var constructorInputs []interface{}
		// Need to figure out the format we need for the constructor arguments
		// Empty array should be fine for verifier contract
		if err != nil {
			fmt.Printf("Error parsing constructor parameters: %s\n", err)
			return
		}


		payload := contract.CompilePayload(binStr, abiStr, constructorInputs...)

		tx, err := contract.DeployContract(
			ctx,
			ethClient.client,
			account.Key.PrivateKey,
			payload,
			nil,
			gasLimit,
		)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("Waiting for contract to be deployed")
		addr, err := bind.WaitDeployed(ctx, ethClient.client, tx)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Deployed contract at: %s\n", addr.String())
}


func transactionMessage(inputArguments []string, contractName string, functionName string, accountName string, contractAddress string, amountString string, gasLimitString string) {

	instance := contracts[contractName]
	methodName := functionName
	account := accounts[accountName]
	contractDeployedAddress := common.HexToAddress(contractAddress)

	if instance == nil {
		errStr := fmt.Sprintf("Contract instance %s not found.\nUse \taddContractInstance [name] [path/to/solidity/contract] \n", contractName)
		fmt.Println(errStr)
		return
	}
	if account == nil {
		errStr := fmt.Sprintf("Account %s not found.\nUse \taddAccount [name] [path/to/keystore]\n", accountName)
		fmt.Println(errStr)
		return
	}
	amount := new(big.Int)
	amount, ok := amount.SetString(amountString, 10)
	if !ok {
		fmt.Printf("Please enter an integer for <amount>")
	}
	gasLimit, err := strconv.ParseUint(gasLimitString, 10, 64)
	if err != nil {
		fmt.Printf("Please enter an integer for <gasLimit>")
	}
	if instance.Abi.Methods[methodName].Name == "" {
		fmt.Printf("Method name \"%s\" not found for contract \"%s\"\n", methodName, contractName)
		return
	}

	inputs, err := customParseMethodParameters(inputArguments, instance.Abi, methodName)
	if err != nil {
		fmt.Printf("Error parsing parameters: %s\n", err)
		return
	}
	for i := 0; i < len(inputs); i++ {
		fmt.Printf("%v", inputs[i])
	}

	tx, err := contract.TransactionContract(
		ctx,
		ethClient.client,
		account.Key.PrivateKey,
		instance.Contract,
		contractDeployedAddress,
		amount,
		gasLimit,
		functionName,
		inputs...,
	)
	if err != nil {
		fmt.Println(err)
		return
	} else {
		fmt.Println("Waiting for transaction to be mined...")
		receipt, err := bind.WaitMined(ctx, ethClient.client, tx)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Transaction hash: %s\n", receipt.TxHash.String())
	}
}


func customParseMethodParameters(argsArray []string, abiStruct *abi.ABI, methodName string) (args []interface{}, err error) {
	var inputParameters abi.Arguments
	if methodName != "" {
		inputParameters = abiStruct.Methods[methodName].Inputs
	} else {
		inputParameters = abiStruct.Constructor.Inputs
	}


	for i := 0; i < len(inputParameters); i++ {
		argument := inputParameters[i]

		input := argsArray[i]

		// bytes = []byte{} argument type = slice, no element, type equates to []uint8
		// byte[] = [][1]byte{} argument type = slice, element type = array, type equates to [][1]uint8
		// byte = bytes1
		// bytesn = [n]byte{} 0 < n < 33, argument type = array, no element, type equates to [n]uint8
		// bytesn[] = [][n]byte{} argument type = slice, element type = array, type equares to [][n]uint8
		// bytesn[m] = [m][n]byte{} argument type = array, element type = array, type equates to [m][n]uint8
		// Many annoying cases of byte arrays

		if argument.Type.Kind == reflect.Array || argument.Type.Kind == reflect.Slice {
			fmt.Println("Argument is array\n")

			// One dimensional byte array
			// Accepts all byte arrays as hex string with pre-pended '0x' only
			if argument.Type.Elem == nil {
				if argument.Type.Type == reflect.TypeOf(common.Address{}) {
					// address solidity type
					item, err := utils.ConvertToType(input, &argument.Type)
					if err != nil {
						fmt.Printf("Error address conversion")
					}
					args = append(args, item)
					continue
				} else if argument.Type.Type == reflect.TypeOf([]byte{}) {
					// bytes solidity type
					bytes, err := hex.DecodeString(input[2:])
					if err != nil {
						fmt.Printf("Error bytes conversion")
					}
					args = append(args, bytes)
					continue
				} else {
					// Fixed byte array of size n; bytesn solidity type
					// Any submitted bytes longer than the expected size will be truncated

					bytes, err := hex.DecodeString(input[2:])
					if err != nil {
						fmt.Printf("Error bytes conversion 2")
					}

					// Fixed sized arrays can't be created with variables as size
					switch argument.Type.Size {
					case 1:
						var byteArray [1]byte
						copy(byteArray[:], bytes[:1])
						args = append(args, byteArray)
					case 2:
						var byteArray [2]byte
						copy(byteArray[:], bytes[:2])
						args = append(args, byteArray)
					case 3:
						var byteArray [3]byte
						copy(byteArray[:], bytes[:3])
						args = append(args, byteArray)
					case 4:
						var byteArray [4]byte
						copy(byteArray[:], bytes[:4])
						args = append(args, byteArray)
					case 5:
						var byteArray [5]byte
						copy(byteArray[:], bytes[:5])
						args = append(args, byteArray)
					case 6:
						var byteArray [6]byte
						copy(byteArray[:], bytes[:6])
						args = append(args, byteArray)
					case 7:
						var byteArray [7]byte
						copy(byteArray[:], bytes[:7])
						args = append(args, byteArray)
					case 8:
						var byteArray [8]byte
						copy(byteArray[:], bytes[:8])
						args = append(args, byteArray)
					case 9:
						var byteArray [9]byte
						copy(byteArray[:], bytes[:9])
						args = append(args, byteArray)
					case 10:
						var byteArray [10]byte
						copy(byteArray[:], bytes[:10])
						args = append(args, byteArray)
					case 11:
						var byteArray [11]byte
						copy(byteArray[:], bytes[:11])
						args = append(args, byteArray)
					case 12:
						var byteArray [12]byte
						copy(byteArray[:], bytes[:12])
						args = append(args, byteArray)
					case 13:
						var byteArray [13]byte
						copy(byteArray[:], bytes[:13])
						args = append(args, byteArray)
					case 14:
						var byteArray [14]byte
						copy(byteArray[:], bytes[:14])
						args = append(args, byteArray)
					case 15:
						var byteArray [15]byte
						copy(byteArray[:], bytes[:15])
						args = append(args, byteArray)
					case 16:
						var byteArray [16]byte
						copy(byteArray[:], bytes[:16])
						args = append(args, byteArray)
					case 17:
						var byteArray [17]byte
						copy(byteArray[:], bytes[:17])
						args = append(args, byteArray)
					case 18:
						var byteArray [18]byte
						copy(byteArray[:], bytes[:18])
						args = append(args, byteArray)
					case 19:
						var byteArray [19]byte
						copy(byteArray[:], bytes[:19])
						args = append(args, byteArray)
					case 20:
						var byteArray [20]byte
						copy(byteArray[:], bytes[:20])
						args = append(args, byteArray)
					case 21:
						var byteArray [21]byte
						copy(byteArray[:], bytes[:21])
						args = append(args, byteArray)
					case 22:
						var byteArray [22]byte
						copy(byteArray[:], bytes[:22])
						args = append(args, byteArray)
					case 23:
						var byteArray [23]byte
						copy(byteArray[:], bytes[:23])
						args = append(args, byteArray)
					case 24:
						var byteArray [24]byte
						copy(byteArray[:], bytes[:24])
						args = append(args, byteArray)
					case 25:
						var byteArray [25]byte
						copy(byteArray[:], bytes[:25])
						args = append(args, byteArray)
					case 26:
						var byteArray [26]byte
						copy(byteArray[:], bytes[:26])
						args = append(args, byteArray)
					case 27:
						var byteArray [27]byte
						copy(byteArray[:], bytes[:27])
						args = append(args, byteArray)
					case 28:
						var byteArray [28]byte
						copy(byteArray[:], bytes[:28])
						args = append(args, byteArray)
					case 29:
						var byteArray [29]byte
						copy(byteArray[:], bytes[:29])
						args = append(args, byteArray)
					case 30:
						var byteArray [30]byte
						copy(byteArray[:], bytes[:30])
						args = append(args, byteArray)
					case 31:
						var byteArray [31]byte
						copy(byteArray[:], bytes[:31])
						args = append(args, byteArray)
					case 32:
						var byteArray [32]byte
						copy(byteArray[:], bytes[:32])
						args = append(args, byteArray)
					default:
						errStr := fmt.Sprintf("Error parsing fixed size byte array. Array of size %d incompatible", argument.Type.Size)
						return nil, errors.New(errStr)
					}
					continue
				}

			}

			array := strings.Split(input, ",")
			argSize := argument.Type.Size
			size := len(array)
			if argSize != 0 {
				for size != argSize {
					fmt.Printf("Please enter %i comma-separated list of elements:\n", argSize)
					/*input = c.ReadLine()
					array = strings.Split(input, ",")
					size = len(array)*/
				}
			}

			size = len(array)

			elementType := argument.Type.Elem

			// Elements cannot be kind slice                                        only mean slice
			if elementType.Kind == reflect.Array && elementType.Type != reflect.TypeOf(common.Address{}) {
				// Is 2D byte array
				/* Nightmare to implement, have to account for:
				   * Slice of fixed byte arrays; bytes32[] in solidity for example, generally bytesn[]
				   * Fixed array of fixed byte arrays; bytes32[10] in solidity for example bytesn[m]
				   * Slice or fixed array of string; identical to above two cases as string in solidity is array of bytes
				   Since the upper bound of elements in an array in solidity is 2^256-1, and each fixed byte array
				   has a limit of bytes32 (bytes1, bytes2, ..., bytes31, bytes32), and Golang array creation takes
				   constant length values, we would have to paste the switch-case containing 1-32 fixed byte arrays
				   2^256-1 times to handle every possibility. Since arrays of arrays in seldom used, we have not
				   implemented it.
				*/

				return nil, errors.New("2D Arrays unsupported. Use \"bytes\" instead.")

				/*
				   slice := make([]interface{}, 0, size)
				   err = addFixedByteArrays(array, elementType.Size, slice)
				   if err != nil {
				       return nil, err
				   }
				   args = append(args, slice)
				   continue
				*/
			} else {
				switch elementType.Type {
				case reflect.TypeOf(bool(false)):
					convertedArray := make([]bool, 0, size)
					for _, item := range array {
						b, err := utils.ConvertToBool(item)
						if err != nil {
							return nil, err
						}
						convertedArray = append(convertedArray, b)
					}
					args = append(args, convertedArray)
				case reflect.TypeOf(int8(0)):
					convertedArray := make([]int8, 0, size)
					for _, item := range array {
						i, err := strconv.ParseInt(item, 10, 8)
						if err != nil {
							return nil, err
						}
						convertedArray = append(convertedArray, int8(i))
					}
					args = append(args, convertedArray)
				case reflect.TypeOf(int16(0)):
					convertedArray := make([]int16, 0, size)
					for _, item := range array {
						i, err := strconv.ParseInt(item, 10, 16)
						if err != nil {
							return nil, err
						}
						convertedArray = append(convertedArray, int16(i))
					}
					args = append(args, convertedArray)
				case reflect.TypeOf(int32(0)):
					convertedArray := make([]int32, 0, size)
					for _, item := range array {
						i, err := strconv.ParseInt(item, 10, 32)
						if err != nil {
							return nil, err
						}
						convertedArray = append(convertedArray, int32(i))
					}
					args = append(args, convertedArray)
				case reflect.TypeOf(int64(0)):
					convertedArray := make([]int64, 0, size)
					for _, item := range array {
						i, err := strconv.ParseInt(item, 10, 64)
						if err != nil {
							return nil, err
						}
						convertedArray = append(convertedArray, int64(i))
					}
					args = append(args, convertedArray)
				case reflect.TypeOf(uint8(0)):
					convertedArray := make([]uint8, 0, size)
					for _, item := range array {
						u, err := strconv.ParseUint(item, 10, 8)
						if err != nil {
							return nil, err
						}
						convertedArray = append(convertedArray, uint8(u))
					}
					args = append(args, convertedArray)
				case reflect.TypeOf(uint16(0)):
					convertedArray := make([]uint16, 0, size)
					for _, item := range array {
						u, err := strconv.ParseUint(item, 10, 16)
						if err != nil {
							return nil, err
						}
						convertedArray = append(convertedArray, uint16(u))
					}
					args = append(args, convertedArray)
				case reflect.TypeOf(uint32(0)):
					convertedArray := make([]uint32, 0, size)
					for _, item := range array {
						u, err := strconv.ParseUint(item, 10, 32)
						if err != nil {
							return nil, err
						}
						convertedArray = append(convertedArray, uint32(u))
					}
					args = append(args, convertedArray)
				case reflect.TypeOf(uint64(0)):
					convertedArray := make([]uint64, 0, size)
					for _, item := range array {
						u, err := strconv.ParseUint(item, 10, 64)
						if err != nil {
							return nil, err
						}
						convertedArray = append(convertedArray, uint64(u))
					}
					args = append(args, convertedArray)
				case reflect.TypeOf(&big.Int{}):
					convertedArray := make([]*big.Int, 0, size)
					for _, item := range array {
						newInt := new(big.Int)
						newInt, ok := newInt.SetString(item, 10)
						if !ok {
							return nil, errors.New("Could not convert string to big.int")
						}
						convertedArray = append(convertedArray, newInt)
					}
					args = append(args, convertedArray)
				case reflect.TypeOf(common.Address{}):
					convertedArray := make([]common.Address, 0, size)
					for _, item := range array {
						a := common.HexToAddress(item)
						convertedArray = append(convertedArray, a)
					}
					args = append(args, convertedArray)
				default:
					errStr := fmt.Sprintf("Type %s not found", elementType.Type)
					return nil, errors.New(errStr)
				}
			}
		} else {
			switch argument.Type.Kind {
			case reflect.String:
				args = append(args, input)
			case reflect.Bool:
				b, err := utils.ConvertToBool(input)
				if err != nil {
					return nil, err
				}
				args = append(args, b)
			case reflect.Int8:
				i, err := strconv.ParseInt(input, 10, 8)
				if err != nil {
					return nil, err
				}
				args = append(args, int8(i))
			case reflect.Int16:
				i, err := strconv.ParseInt(input, 10, 16)
				if err != nil {
					return nil, err
				}
				args = append(args, int16(i))
			case reflect.Int32:
				i, err := strconv.ParseInt(input, 10, 32)
				if err != nil {
					return nil, err
				}
				args = append(args, int32(i))
			case reflect.Int64:
				i, err := strconv.ParseInt(input, 10, 64)
				if err != nil {
					return nil, err
				}
				args = append(args, int64(i))
			case reflect.Uint8:
				u, err := strconv.ParseUint(input, 10, 8)
				if err != nil {
					return nil, err
				}
				args = append(args, uint8(u))
			case reflect.Uint16:
				u, err := strconv.ParseUint(input, 10, 16)
				if err != nil {
					return nil, err
				}
				args = append(args, uint16(u))
			case reflect.Uint32:
				u, err := strconv.ParseUint(input, 10, 32)
				if err != nil {
					return nil, err
				}
				args = append(args, uint32(u))
			case reflect.Uint64:
				u, err := strconv.ParseUint(input, 10, 64)
				if err != nil {
					return nil, err
				}
				args = append(args, uint64(u))
			case reflect.Ptr:
				newInt := new(big.Int)
				newInt, ok := newInt.SetString(input, 10)
				if !ok {
					return nil, errors.New("Could not convert string to big.int")
				}
				if err != nil {
					return nil, err
				}
				args = append(args, newInt)
			case reflect.Array:
				if argument.Type.Type == reflect.TypeOf(common.Address{}) {
					address := common.HexToAddress(input)
					args = append(args, address)
				} else {
					return nil, errors.New("Conversion failed. Item is array type, cannot parse")
				}
			default:
				errStr := fmt.Sprintf("Error, type not found: %s", argument.Type.Kind)
				return nil, errors.New(errStr)
			}
		}
	}
	return
}
