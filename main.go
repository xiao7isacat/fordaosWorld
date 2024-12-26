package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// 代币合约地址
var (
	ARContractAddress    = common.HexToAddress("0x...")
	AISTRContractAddress = common.HexToAddress("0x...")
	ALCHContractAddress  = common.HexToAddress("0x...")
)

// 代币合约的 ABI
const tokenABI = `[{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"type":"function"}]`

// 获取代币余额
func getTokenBalance(client *ethclient.Client, contractAddress common.Address, holderAddress common.Address, blockNumber *big.Int) (*big.Int, error) {
	parsedABI, err := abi.JSON(strings.NewReader(tokenABI))
	if err != nil {
		return nil, err
	}

	callData, err := parsedABI.Pack("balanceOf", holderAddress)
	if err != nil {
		return nil, err
	}

	msg := ethereum.CallMsg{
		To:   &contractAddress,
		Data: callData,
	}

	result, err := client.CallContract(context.Background(), msg, blockNumber)
	if err != nil {
		return nil, err
	}

	balance := new(big.Int)
	balance.SetBytes(result)
	return balance, nil
}

// 获取所有持有者的代币余:
func getHoldersBalances(client *ethclient.Client, blockNumber *big.Int, holderAddresses []common.Address) map[common.Address]map[string]*big.Int {
	holders := make(map[common.Address]map[string]*big.Int)

	for _, address := range holderAddresses {
		arBalance, _ := getTokenBalance(client, ARContractAddress, address, blockNumber)
		aistrBalance, _ := getTokenBalance(client, AISTRContractAddress, address, blockNumber)
		alchBalance, _ := getTokenBalance(client, ALCHContractAddress, address, blockNumber)

		totalBalance := new(big.Int).Add(arBalance, aistrBalance)
		totalBalance.Add(totalBalance, alchBalance)

		if totalBalance.Cmp(big.NewInt(0)) > 0 {
			holders[address] = map[string]*big.Int{
				"AR":    arBalance,
				"AISTR": aistrBalance,
				"ALCH":  alchBalance,
			}
		}
	}

	return holders
}

// 加权随机选择
func weightedRandomSelection(holders map[common.Address]map[string]*big.Int, numWinners int) []common.Address {
	var winners []common.Address
	var totalWeight *big.Int = big.NewInt(0)
	weights := make(map[common.Address]*big.Int)

	for address, balances := range holders {
		weight := new(big.Int).Add(balances["AR"], balances["AISTR"])
		weight.Add(weight, balances["ALCH"])
		weights[address] = weight
		totalWeight.Add(totalWeight, weight)
	}

	rand.Seed(time.Now().UnixNano())
	for len(winners) < numWinners {
		r := new(big.Int).Rand(rand.New(rand.NewSource(time.Now().UnixNano())), totalWeight)
		var cumulativeWeight *big.Int = big.NewInt(0)

		for address, weight := range weights {
			cumulativeWeight.Add(cumulativeWeight, weight)
			if r.Cmp(cumulativeWeight) < 0 {
				winners = append(winners, address)
				break
			}
		}
	}

	return winners
}

func main() {
	client, err := ethclient.Dial("https://mainnet.infura.io/v3/YOUR_INFURA_PROJECT_ID")
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}

	blockNumber := big.NewInt(12345678) // 使用特定的区块编号作为快照
	holderAddresses := []common.Address{
		// 在这里添加持有者地址
	}

	holders := getHoldersBalances(client, blockNumber, holderAddresses)
	winners := weightedRandomSelection(holders, 100)

	for _, winner := range winners {
		fmt.Printf("Address: %s, Balances: AR=%s, AISTR=%s, ALCH=%s\n",
			winner.Hex(),
			holders[winner]["AR"].String(),
			holders[winner]["AISTR"].String(),
			holders[winner]["ALCH"].String())
	}
}
