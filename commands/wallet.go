/*
Copyright 2018 The Dccncli Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Ankr-network/ankrctl/types"
	"github.com/Ankr-network/ankrctl/commands/displayers"
	ankr_const "github.com/Ankr-network/dccn-common"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	gwusermgr "github.com/Ankr-network/dccn-common/protos/gateway/usermgr/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"github.com/Ankr-network/ankr-chain/common"
	"github.com/Ankr-network/ankr-chain/tx/serializer"
	"github.com/Ankr-network/ankr-chain/client"
	"github.com/Ankr-network/ankr-chain/tx/token"
	"github.com/Ankr-network/ankr-chain/crypto"
)

var tendermintURL = "https://chain-01.dccn.ankr.com;https://chain-02.dccn.ankr.com;https://chain-03.dccn.ankr.com"
var ankrChainId = "ankr-chain"
var tendermintPort = "443"
var ankrCurrency = common.Currency{"ANKR",18}
var ankrGasLimit = big.NewInt(20000)

// walletCmd creates the wallet command.
func walletCmd() *Command {
	//DCCN-CLI wallet
	cmd := &Command{
		Command: &cobra.Command{
			Use:     "wallet",
			Aliases: []string{"w"},
			Short:   "wallet commands",
			Long:    "wallet is used to access wallet commands",
		},
		DocCategories: []string{"wallet"},
		IsIndex:       true,
	}

	//DCCN-CLI wallet genkey
	cmdWalletGenkey := CmdBuilder(cmd, RunWalletGenkey, "genkey <keyname>", "generate key pair for Mainnet",
		Writer, aliasOpt("gk"), docCategories("wallet"))
	_ = cmdWalletGenkey

	//DCCN-CLI wallet keylist
	cmdWalletKeylist := CmdBuilder(cmd, RunWalletKeylist, "listkey", "list key pair for Mainnet",
		Writer, aliasOpt("kl"), docCategories("wallet"))
	_ = cmdWalletKeylist

	//DCCN-CLI wallet importkey
	cmdWalletImportkey := CmdBuilder(cmd, RunWalletImportkey, "importkey <keyname>",
		"import key from keyfile", Writer, aliasOpt("ik"), docCategories("wallet"))
	AddStringFlag(cmdWalletImportkey, types.ArgKeyFileSlug, "", "", "wallet keyfile", requiredOpt())

	//DCCN-CLI wallet deletekey
	cmdWalletDeletekey := CmdBuilder(cmd, RunWalletDeletekey, "deletekey <keyname>",
		"delete key", Writer, aliasOpt("dk"), docCategories("wallet"))
	_ = cmdWalletDeletekey

	//DCCN-CLI wallet send coins
	cmdWalletSendCoins := CmdBuilder(cmd, RunWalletSendCoins, "sendcoins <symbol>",
		"send token to address", Writer, aliasOpt("st"), docCategories("wallet"))
	AddStringFlag(cmdWalletSendCoins, types.ArgTargetAddressSlug, "", "", "send token to wallet address",
		requiredOpt())
	AddStringFlag(cmdWalletSendCoins, types.ArgKeyFileSlug, "", "", "wallet keyfile", requiredOpt())
	AddStringFlag(cmdWalletSendCoins, types.ArgTxAmount, "", "", "transfer amount", requiredOpt())
	AddStringFlag(cmdWalletSendCoins, types.ArgTxMemo, "", "", "transaction memo", )
	AddStringFlag(cmdWalletSendCoins, types.ArgGasPrice, "", "10000000000000000", "gas price of the transaction", )
	AddStringFlag(cmdWalletSendCoins, types.ArgTxVersion, "", "1.0", "ankr chain version", )


	//DCCN-CLI wallet get balance
	cmdWalletGetbalance := CmdBuilder(cmd, RunWalletGetbalance, "getbalance <address>",
		"get balance of wallet by address", Writer, aliasOpt("gb"), docCategories("wallet"))
	_ = cmdWalletGetbalance

	//DCCN-CLI wallet generate erc address
	cmdWalletGenAddress := CmdBuilder(cmd, RunWalletGenAddress, "genaddr",
		"generate wallet address for deposit and withdraw", Writer, aliasOpt("ga"), docCategories("wallet"))
	AddStringFlag(cmdWalletGenAddress, types.ArgAddressTypeSlug, "", "", "wallet address type (MAINNET/ERC20/BEP2)", requiredOpt())
	AddStringFlag(cmdWalletGenAddress, types.ArgAddressPurposeSlug, "", "", "wallet address purpose (MAINNET/ERC20/BEP2)", requiredOpt())

	//DCCN-CLI wallet search deposit in a period
	cmdWalletSearchDeposit := CmdBuilder(cmd, RunWalletSearchDeposit, "search",
		"wallet search deposit in a period", Writer, aliasOpt("sd"), docCategories("wallet"))
	AddStringFlag(cmdWalletSearchDeposit, types.ArgSearchDepositStartSlug, "", "", "wallet search deposit start date (format: `mm/dd/yyyy`)", requiredOpt())
	AddStringFlag(cmdWalletSearchDeposit, types.ArgSearchDepositEndSlug, "", "", "wallet address deposit end date (format: `mm/dd/yyyy`)", requiredOpt())

	//DCCN-CLI wallet get deposit history
	cmdWalletDepositHistory := CmdBuilder(cmd, RunWalletDepositHistory, "history",
		"retrieve wallet deposit history", Writer, aliasOpt("dh"), docCategories("wallet"))
	_ = cmdWalletDepositHistory

	return cmd

}

// RunWalletGenkey generate wallet key.
func RunWalletGenkey(c *CmdConfig) error {

	if len(c.Args) < 1 {
		return types.NewMissingArgsErr(c.NS)
	}

	files, err := ioutil.ReadDir(configHome())
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}

	path := configHome() + "/"

	for _, f := range files {
		matched, err := filepath.Match("UTC*", f.Name())
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
			return nil
		}
		if matched {
			kf, err := ioutil.ReadFile(path + f.Name())
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
				return nil
			}
			var key EncryptedKeyJSONV3
			err = json.Unmarshal(kf, &key)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
				return nil
			}
			if key.Name == c.Args[0] {
				fmt.Fprintf(os.Stderr, "\nERROR: %s\n", "key name already exists")
				return nil
			}
		}
	}

	if AskForConfirm(fmt.Sprintf(`please record and backup keystore once it is generated, we don’t store your private key! 
	 type 'yes' to confirm that you understood the result of this action: `)) == nil {

		fmt.Println("\ngenerating keys...")

		//privateKey, publicKey, address := wallet.GenerateKeys()
		privateKey, address := GenAccount()

		if privateKey == "" || address == "" {
			fmt.Fprintf(os.Stderr, "generated keys error: got empty secrets")
			return nil
		}

		fmt.Println("private key: ", privateKey, "\naddress: ", address)

		fmt.Print("\nabout to export to keystore...\nplease input the keystore encryption password: ")
		password, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
			return nil
		}

		fmt.Print("\nplease input password again: ")
		confirmPassword, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
			return nil
		}

		if string(password) != string(confirmPassword) {
			fmt.Fprintf(os.Stderr, "\nERROR: %s\n", "password and confirm password not match")
			return nil
		}

		cryptoStruct, err := EncryptDataV3([]byte(privateKey), []byte(password), StandardScryptN, StandardScryptP)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
			return nil
		}

		encryptedKeyJSONV3 := EncryptedKeyJSONV3{
			Name:c.Args[0],
			Address:address,
			Crypto:cryptoStruct,
			KeyJSONVersion:keyJSONVersion,
		}

		jsonKey, err := json.Marshal(encryptedKeyJSONV3)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
			return nil
		}

		fmt.Println("\n\nexporting to keystore...")

		ts := time.Now().UTC()

		kfw, err := KeyFileWriter(fmt.Sprintf("UTC--%s--%s", toISO8601(ts), address))
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
			return nil
		}

		defer kfw.Close()

		_, err = kfw.Write(jsonKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nERROR: %s\n", "unable to write keystore")
		}

		fmt.Fprintf(os.Stderr, "\ncreated keystore: %s/UTC--%s--%s\n\n", configHome(), toISO8601(ts), address)

	}

	return nil
}

// RunWalletKeylist list key in $HOME/.ankr
func RunWalletKeylist(c *CmdConfig) error {

	files, err := ioutil.ReadDir(configHome())
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}

	path := configHome() + "/"

	var keylist []*displayers.KeyStore

	for _, f := range files {
		matched, err := filepath.Match("UTC*", f.Name())
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
			return nil
		}
		if matched {
			kf, err := ioutil.ReadFile(path + f.Name())
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
				return nil
			}
			var key EncryptedKeyJSONV3
			err = json.Unmarshal(kf, &key)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
				return nil
			}
			keylist = append(keylist, &displayers.KeyStore{
				Name:      key.Name,
				Address:   key.Address,
				PublicKey: key.PublicKey,
			})
		}
	}
	item := &displayers.Key{Keystores: keylist}
	return c.Display(item)
}

// RunWalletImportkey import wallet key.
func RunWalletImportkey(c *CmdConfig) error {

	if len(c.Args) < 1 {
		return types.NewMissingArgsErr(c.NS)
	}

	ks, err := c.Ankr.GetString(c.NS, types.ArgKeyFileSlug)
	if err != nil {
		return err
	}

	kf, err := ioutil.ReadFile(ks)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}

	var key EncryptedKeyJSONV3

	err = json.Unmarshal(kf, &key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}

	files, err := ioutil.ReadDir(configHome())
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}
	path := configHome() + "/"

	for _, f := range files {
		matched, err := filepath.Match("UTC*", f.Name())
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
			return nil
		}
		if matched {
			kf, err := ioutil.ReadFile(path + f.Name())
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
				return nil
			}
			var ks EncryptedKeyJSONV3
			err = json.Unmarshal(kf, &ks)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
				return nil
			}
			if ks.Name == c.Args[0] {
				fmt.Fprintf(os.Stderr, "\nERROR: key '%s' already exists.\n", ks.Name)
				return nil
			}
		}
	}

	fmt.Print("please input the keystore password: ")
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}
	_, err = DecryptDataV3(key.Crypto, string(password))
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}

	key.Name = c.Args[0]
	ts := time.Now().UTC()
	jsonKey, err := json.Marshal(key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}
	kfw, err := KeyFileWriter(fmt.Sprintf("UTC--%s--%s", toISO8601(ts), key.Address))
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}
	defer kfw.Close()

	_, err = kfw.Write(jsonKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", "unable to write keystore")
		return nil
	}

	fmt.Printf("\nkeystore imported: %s/UTC--%s--%s\n\n", configHome(), toISO8601(ts), key.Address)

	return nil
}

// RunWalletDeletekey delete wallet key.
func RunWalletDeletekey(c *CmdConfig) error {

	if len(c.Args) < 1 {
		return types.NewMissingArgsErr(c.NS)
	}

	if AskForConfirm(fmt.Sprintf(`about to delete keystore '%s', type 'yes' to confirm, 'no' to cancel: `, c.Args[0])) == nil {

		files, err := ioutil.ReadDir(configHome())
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
			return nil
		}
		path := configHome() + "/"

		for _, f := range files {
			matched, err := filepath.Match("UTC*", f.Name())
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
				return nil
			}
			if matched {
				kf, err := ioutil.ReadFile(path + f.Name())
				if err != nil {
					fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
					return nil
				}
				var ks EncryptedKeyJSONV3
				err = json.Unmarshal(kf, &ks)
				if err != nil {
					fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
					return nil
				}
				if ks.Name == c.Args[0] {
					err := os.Remove(path + f.Name())
					if err != nil {
						fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
					} else {
						fmt.Fprintf(os.Stderr, "\nkeystore '%s' deleted\n", c.Args[0])
					}
					return nil
				}
			}
		}
		fmt.Fprintf(os.Stderr, "\nERROR: no keystore found with name '%s'\n", c.Args[0])
	}

	return nil
}

// RunWalletSendtoken send token to other wallet address.
func RunWalletSendCoins(c *CmdConfig) error {

	if len(c.Args) < 1 {
		return types.NewMissingArgsErr(c.NS)
	}

	target, err := c.Ankr.GetString(c.NS, types.ArgTargetAddressSlug)
	if err != nil {
		return err
	}

	//amount := c.Args[0]
	txSymbol := c.Args[0]
	amount, err := c.Ankr.GetString(c.NS, types.ArgTxAmount)
	if err != nil {
		return err
	}

	tokenAmount, ok := new(big.Int).SetString(amount, 10)
	if !ok {
		fmt.Fprintf(os.Stderr, "\nERROR: can not parsing amount '%s'\n", amount)
		return nil
	}
	msgCur := common.Currency{Symbol:txSymbol}
	msgAmount := common.Amount{msgCur, tokenAmount.Bytes()}

	txVersion, err := c.Ankr.GetString(c.NS, types.ArgTxVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}

	txMemo, err := c.Ankr.GetString(c.NS, types.ArgTxMemo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}
	txGasPrice, err := c.Ankr.GetString(c.NS, types.ArgGasPrice)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}
	gasPriceInt, ok := new(big.Int).SetString(txGasPrice, 10)
	if !ok {
		fmt.Fprintf(os.Stderr, "\nERROR: can not parsing amount '%s'\n", amount)
		return nil
	}
	gasPrice := new(common.Amount)
	gasPrice.Cur = ankrCurrency
	gasPrice.Value = gasPriceInt.Bytes()


	keystore, err := c.Ankr.GetString(c.NS, types.ArgKeyFileSlug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}
	ksf := keystore

	files, err := ioutil.ReadDir(configHome())
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}

	path := configHome() + "/"

	for _, f := range files {
		matched, err := filepath.Match("UTC*", f.Name())
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
			return nil
		}
		if matched {
			kf, err := ioutil.ReadFile(path + f.Name())
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
				return nil
			}
			var key EncryptedKeyJSONV3
			err = json.Unmarshal(kf, &key)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
				return nil
			}
			if key.Name == keystore {
				ksf = configHome() + "/" + f.Name()
				break
			}
		}
	}

	ks, err := ioutil.ReadFile(ksf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}

	var key EncryptedKeyJSONV3

	err = json.Unmarshal(ks, &key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}
	address := key.Address

	fmt.Print("please input the keystore password: ")
	password, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}
	privateKeyDecrypt, err := DecryptDataV3(key.Crypto, string(password))
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}
	privateKey := string(privateKeyDecrypt)

	if len(address) == 0 || len(privateKey) == 0 {

		fmt.Println("\nno key found, please sign this transaction with your wallet address and private key")

		fmt.Print("\naddress: ")
		address, err = retrieveUserInput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
			return nil
		}

		fmt.Print("\nprivate key: ")
		privateKey, err = retrieveUserInput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
			return nil
		}
	}
	fmt.Println("")
	if len(address) == 0 || len(privateKey) == 0 {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", "empty wallet address or private key")
		return nil
	}

	txkey := crypto.NewSecretKeyEd25519(privateKey)
	if len(tendermintURL) == 0 {
		tendermintURL = "https://chain-01.dccn.ankr.com;https://chain-02.dccn.ankr.com;https://chain-03.dccn.ankr.com"
	}
	if len(tendermintPort) == 0 {
		tendermintPort = "443"
	}
	fmt.Println("")
	if AskForConfirm(fmt.Sprintf("about to send %s tokens from address '%s' to address '%s', type 'yes' to confirm this action: ", c.Args[0], address, target)) == nil {
		urls := strings.Split(tendermintURL, ";")
		randomUrls := Shuffle(urls)
		tendermintURL = randomUrls[0]
		for _, url := range randomUrls {
			netinfoURL := url + ":" + tendermintPort + "/net_info"
			rsp, err := http.Get(netinfoURL)
			if err == nil && rsp.StatusCode == 200 {
				tendermintURL = url
				break
			}
		}

		//start sending transaction
		fmt.Fprintf(os.Stderr, "\nsending %s %s from address '%s' to address '%s'\n", tokenAmount.String(), txSymbol, address, target)

		//fill transaction meta data into msg
		txMsgHeader := new(client.TxMsgHeader)
		txMsgHeader.ChID = common.ChainID(ankrChainId)
		txMsgHeader.GasLimit = ankrGasLimit.Bytes()
		txMsgHeader.Version = txVersion
		txMsgHeader.Memo = txMemo
		txMsgHeader.GasPrice = *gasPrice

		transferMsg := new(token.TransferMsg)
		transferMsg.FromAddr = address
		transferMsg.ToAddr = target
		transferMsg.Amounts = append(transferMsg.Amounts, msgAmount)

		//start sending transactions
		cl := client.NewClient(tendermintURL+":"+tendermintPort)
		builder := client.NewTxMsgBuilder(*txMsgHeader, transferMsg, serializer.NewTxSerializerCDC(), txkey)
		txHash, txHeight, _, err := builder.BuildAndCommit(cl)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
			return nil
		}
		fmt.Fprintf(os.Stderr, "\nTransaction commit success.")
		fmt.Fprintf(os.Stderr, "\ntx hash: %s\n", txHash)
		fmt.Fprintf(os.Stderr, "\ntx height: %d\n", txHeight)
	}
	return nil
}

// RunWalletGetbalance get balance from chain.
func RunWalletGetbalance(c *CmdConfig) error {

	if len(c.Args) < 1 {
		return types.NewMissingArgsErr(c.NS)
	}
	address := c.Args[0]

	fmt.Printf("\nquerying balance of address: %s\n", address)
	if tendermintURL == "" {
		tendermintURL = "https://chain-01.dccn.ankr.com;https://chain-02.dccn.ankr.com;https://chain-03.dccn.ankr.com"
	}
	if tendermintPort == "" {
		tendermintPort = "443"
	}

	urls := strings.Split(tendermintURL, ";")
	randomUrls := Shuffle(urls)
	tendermintURL = randomUrls[0]
	for _, url := range randomUrls {
		netinfoURL := url + ":" + tendermintPort + "/net_info"
		rsp, err := http.Get(netinfoURL)
		if err == nil && rsp.StatusCode == 200 {
			tendermintURL = url
			break
		}
	}

	cl := client.NewClient(tendermintURL+":"+tendermintPort)
	balReq := new(common.BalanceQueryReq)
	balResp := new(common.BalanceQueryResp)
	balReq.Symbol = "ANKR"
	balReq.Address = address
	err := cl.Query("/store/balance", balReq, balResp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}

	fmt.Printf("wallet balance: %s\n", balResp.Amount)

	return nil
}

// RunWalletGenAddress generate wallet key for deposit/withdraw.
func RunWalletGenAddress(c *CmdConfig) error {

	s := map[string]bool{"MAINNET": true, "ERC20": true, "BEP2": true}

	addressType, err := c.Ankr.GetString(c.NS, types.ArgAddressTypeSlug)
	if err != nil {
		return err
	}

	addressPurpose, err := c.Ankr.GetString(c.NS, types.ArgAddressPurposeSlug)
	if err != nil {
		return err
	}

	_, typeOk := s[addressType]
	_, purposeOk := s[addressPurpose]

	if !typeOk || !purposeOk {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", "type or purpose not one of MAINNET/ERC20/BEP2..")
		return nil
	}

	authResult := gwusermgr.AuthenticationResult{}
	viper.UnmarshalKey("AuthResult", &authResult)

	if len(authResult.AccessToken) == 0 {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", "no ankr network access token found")
		return nil
	}

	md := metadata.New(map[string]string{
		"token": authResult.AccessToken,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenctx, cancel := context.WithTimeout(ctx, ankr_const.ClientTimeOut*time.Second)
	defer cancel()

	url := viper.GetString("hub-url")

	conn, err := grpc.Dial(url+port, grpc.WithInsecure())
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: did not connect, %v \n", err)
		return nil
	}

	defer conn.Close()
	userClient := gwusermgr.NewUserMgrClient(conn)

	rsp, err := userClient.CreateAddress(tokenctx,
		&gwusermgr.GenerateAddressRequest{
			Type:    addressType,
			Purpose: addressPurpose,
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}

	fmt.Fprintf(os.Stderr, "\ngenerated address type %s '%s' for purpose %s \n",
		addressType, rsp.Typeaddress, addressPurpose)

	return nil
}

// RunWalletSearchDeposit search deposit for certain period.
func RunWalletSearchDeposit(c *CmdConfig) error {

	start, err := c.Ankr.GetString(c.NS, types.ArgSearchDepositStartSlug)
	if err != nil {
		return err
	}
	startTime, err := time.Parse("01/02/2006", start)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}
	end, err := c.Ankr.GetString(c.NS, types.ArgSearchDepositEndSlug)
	if err != nil {
		return err
	}
	endTime, err := time.Parse("01/02/2006", end)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}

	authResult := gwusermgr.AuthenticationResult{}
	viper.UnmarshalKey("AuthResult", &authResult)

	if len(authResult.AccessToken) == 0 {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", "no ankr network access token found")
		return nil
	}

	md := metadata.New(map[string]string{
		"token": authResult.AccessToken,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenctx, cancel := context.WithTimeout(ctx, ankr_const.ClientTimeOut*time.Second)
	defer cancel()

	url := viper.GetString("hub-url")

	conn, err := grpc.Dial(url+port, grpc.WithInsecure())
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: did not connect, %v\n", err)
		return nil
	}

	defer conn.Close()
	userClient := gwusermgr.NewUserMgrClient(conn)

	rsp, err := userClient.SearchDeposit(tokenctx,
		&gwusermgr.SearchDepositRequest{
			Start: &timestamp.Timestamp{
				Seconds: startTime.Unix(),
			},
			End: &timestamp.Timestamp{
				Seconds: endTime.Unix(),
			},
		})
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}

	for _, v := range rsp.Deposits {
		amount, ok := new(big.Float).SetString(v.Amount)
		if !ok {
			fmt.Fprintf(os.Stderr, "\nERROR: can not parse amount '%s'\n", v.Amount)
			return nil
		}
		fmt.Printf("Time: %s\nHash: %s\nState: %s\nConfirmed Block Height: %s\nFrom Account Address Type: %s\nFrom Account Address: %s\nTo Account Address Type: %s\nTo Account Address: %s\nAmount: %v\n\n",
			ptypes.TimestampString(v.Time), v.TxHash, v.TxState, v.ConfirmedBlockHeight, v.FromAccountAddressType, v.FromAccountAddress, v.ToAccountAddressType, v.ToAccountAddress, new(big.Float).Quo(amount, big.NewFloat(float64(1000000000000000000.0))).String())
	}

	return nil
}

// RunWalletDepositHistory return deposit history for certain period.
func RunWalletDepositHistory(c *CmdConfig) error {

	authResult := gwusermgr.AuthenticationResult{}
	viper.UnmarshalKey("AuthResult", &authResult)

	if len(authResult.AccessToken) == 0 {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", "no ankr network access token found")
		return nil
	}

	md := metadata.New(map[string]string{
		"token": authResult.AccessToken,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenctx, cancel := context.WithTimeout(ctx, ankr_const.ClientTimeOut*time.Second)
	defer cancel()

	url := viper.GetString("hub-url")

	conn, err := grpc.Dial(url+port, grpc.WithInsecure())
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: did not connect: %v\n", err)
		return nil
	}

	defer conn.Close()
	userClient := gwusermgr.NewUserMgrClient(conn)

	rsp, err := userClient.DepositHistory(tokenctx, &common_proto.Empty{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nERROR: %s\n", err.Error())
		return nil
	}

	for _, v := range rsp.Deposits {
		amount, ok := new(big.Float).SetString(v.Amount)
		if !ok {
			fmt.Fprintf(os.Stderr, "\nERROR: can not parse amount '%s'\n", v.Amount)
			return nil
		}
		fmt.Fprintf(os.Stderr, "Time: %s\nHash: %s\nState: %s\nConfirmed Block Height: %s\nFrom Account Address Type: %s\nFrom Account Address: %s\nTo Account Address Type: %s\nTo Account Address: %s\nAmount: %v\n\n",
			ptypes.TimestampString(v.Time), v.TxHash, v.TxState, v.ConfirmedBlockHeight, v.FromAccountAddressType, v.FromAccountAddress, v.ToAccountAddressType, v.ToAccountAddress, new(big.Float).Quo(amount, big.NewFloat(float64(1000000000000000000.0))).String())
	}

	return nil
}

func Shuffle(slice []string) []string {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	ret := make([]string, len(slice))
	n := len(slice)
	for i := 0; i < n; i++ {
		randIndex := r.Intn(len(slice))
		ret[i] = slice[randIndex]
		slice = append(slice[:randIndex], slice[randIndex+1:]...)
	}
	return ret
}
