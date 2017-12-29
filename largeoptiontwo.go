package main
import (
        "encoding/json"
        "github.com/hyperledger/fabric/common/util"
		"fmt"
		"math"
        "strings"
		"strconv" 
		"github.com/hyperledger/fabric/core/chaincode/shim"
		pb "github.com/hyperledger/fabric/protos/peer"
)
const success="success"
// channelId:="mychannel"
// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}
//持仓表
type SSEHold struct{
	ObjectType string `json:"docType"`
	AccId string `json:"accId"`
	ProductCode string `json:"productCode"`
	HoldNum int `json:"holdNum"`
	FrozenSecNum int `json:"frozenSecNum"`
}
type AcctAsset struct{
	ObjectType string `json:"docType"`
	AcctId string `json:"acctId"`
	AvaMoney int `json:"avaMoney"`
}
type ContractHold struct{
    ObjectType string `json:"docType"`
    ContractN string `json:"contractN"`
    ContractCode string `json:"contractCode"`
    ContractStatus string `json:"contractStatus"`//"0"代表等待开始，"1"代表开始合约，"2"代表已经开始,
//"3"代表行权，"4"代表已终结,"5"代表互操作,"6"代表待联通,"7"代表联通,"8"代表解质押
    ContractCCID string `json:"contractCCID"`//合约名
    ContractFunctionName string `json:"contractFunctionName"`
    AccIdA string `json:"accIdA"`//合约关联方A
    AccIdB string `json:"accIdB"`
    AccIdC string `json:"accIdC"`
    AccIdD string `json:"accIdD"`
    AccIdE string `json:"accIdE"`
    CorContractNA string `json:"corContractNA"`//关联合约
    CorContractNB string `json:"corContractNB"`
    CorContractNC string `json:"corContractNC"`
    TransType string `json:"transType"`//"0"代表不可转让，"1"代表可转让
    LastSwapTime string `json:"lastSwapTime"`
}
//账户流水表
type AccFlow struct{
	ObjectType string `json:"docType"`
	AccFlowId string `json:"accFlowId"`//账户流水编号
	AccId string `json:"accId"`
	AssetId string `json:"assetId"`
	AssetNum string `json:"assetNum"`
	SType string `json:"sType"`//"0"代表增加,"1"代表减少,"2"代表锁定,"3"代表解锁
	ContractN string `json:"contractN"`
	Time string `json:"time"`
}
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response  {
        fmt.Println("########### example_cc Init ###########")
	
	return shim.Success(nil)
}
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface) pb.Response {
		return shim.Error("Unknown supported call")
}
// Transaction makes payment of X units from A to B
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
        fmt.Println("########### example_cc Invoke ###########")
	function, args := stub.GetFunctionAndParameters()
	
	if function != "invoke" {
                return shim.Error("Unknown function call")
	}

	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting at least 2")
	}
	//设计合约
	if args[0]=="bigoptiontwo"{
		return t.bigoptiontwo(stub,args)
	}
	return shim.Error("Unknown action, check the first argument, must be one of 'delete', 'query', or 'move'")
}
//设计合约
//bigDealOne,contractN,actionType,accId,pw,accIdN,corContractNAHoldStr,
//(accFlowId,acctAssetA,acctAssetB),channelId,chaincodeToCall,
func (t *SimpleChaincode) bigoptiontwo(stub shim.ChaincodeStubInterface,args []string)pb.Response{
	if len(args)!=10{
		return shim.Error("args numbers is wrong")
	}
	if args[1]==""||args[2]==""||args[3]==""||args[4]==""||args[5]==""{
		return shim.Error("args can not be nil")
	}
	contractNStr:=args[1]
	actionType:=args[2]
	accIdStr:=args[3]
	pwStr:=args[4]
	accIdNStr:=args[5]
    corContractNAHoldStr:=args[6]
    allData:=args[7]
	channelId:=args[8]
	chaincodeToCall:=args[9]
	//账户流水
	var accFlowAAsset,accFlowBAsset AccFlow
    //账户流水编号
    var accFlowIdStr string
	//获得合约持仓
    contractHold,result:=getContractHold(stub,contractNStr,chaincodeToCall,channelId)
    if result!=success{
    	return shim.Error(result)
    }
    //资金账户
    var acctAssetA,acctAssetB AcctAsset
    isTermination:=true
    if corContractNAHoldStr!=""{
        var corContractHoldA ContractHold
        err:=json.Unmarshal([]byte(corContractNAHoldStr),&corContractHoldA)
        if err!=nil{
            return shim.Error("corContractHoldA Unmarshal failed")
        }
        //解析参数
        //(accFlowId,acctAssetA)
        dataArr:=strings.Split(allData,";")
        accFlowIdStr=dataArr[0]
        accFlowId,err:=strconv.Atoi(accFlowIdStr)
        if err!=nil{
            return shim.Error("accFlowIdStr Atoi failed")
        }
        //传进来的A变成现在的B
        err=json.Unmarshal([]byte(dataArr[1]),&acctAssetB)
        if err!=nil{
            return shim.Error("acctAssetB Unmarshal failed")
        }
        //只用到了一个参数
        if len(dataArr)==3{
            result=saveSSEHoldByAccAndProduct(stub,dataArr[2],acctAssetB.AcctId,"sh0003",chaincodeToCall,channelId)
            if result!=success{
                return shim.Error(result)
            }
        }
        //获得账户A的资金账户
        acctAssetA,result=getAcctAsset(stub,contractHold.AccIdA,chaincodeToCall,channelId)
        if result!=success{
            return shim.Error(result)
        }
        if actionType=="7"&&corContractHoldA.ContractStatus=="6"{
            if contractHold.ContractStatus=="0"{
                //判断A是否有足够的钱
                if acctAssetA.AvaMoney<5000{
                    return shim.Error("acctA do not have enough money")
                }
                contractHold.ContractStatus="6"//待联通
                //A给B转钱
                acctAssetA.AvaMoney-=5000
                acctAssetB.AvaMoney+=5000
                if acctAssetB.AvaMoney<5000{
                    return shim.Error("acctB do not have enough money")
                }
                timeStr:=getCurrTime(stub,chaincodeToCall,channelId)
                //账户A的流水
                accFlowAAsset=AccFlow{
                    "accFlow",
                    accFlowIdStr,
                    contractHold.AccIdA,
                    "money",
                    "5000",
                    "1",
                    contractNStr,
                    timeStr,
                }
                //账户B的流水
                accFlowId=accFlowId+1
                accFlowIdStr=strconv.Itoa(accFlowId)
                accFlowBAsset=AccFlow{
                    "accFlow",
                    accFlowIdStr,
                    contractHold.AccIdB,
                    "money",
                    "5000",
                    "0",
                    contractNStr,
                    timeStr,
                }
                contractHold.ContractStatus="6"
                if contractHold.CorContractNB!=""{
                    isTermination=false
                    accFlowId=accFlowId+1
                    accFlowIdStr=strconv.Itoa(accFlowId)
                    corContractHoldB,result:=getContractHold(stub,contractHold.CorContractNB,chaincodeToCall,channelId)
                    if result!=success{
                        return shim.Error(result)
                    }
                    //构造调用参数
                    contractHoldBytes,err:=json.Marshal(contractHold)
                    if err!=nil{
                        return shim.Error("contractHoldBytes Marshal failed")
                    }
                    acctAssetABytes,err:=json.Marshal(acctAssetA)
                    if err!=nil{
                        return shim.Error("acctAssetABytes Marshal failed")
                    }
                    //bigdealtwo传过来三个参数，第三个参数在该合约中没有用到
                    allData=accFlowIdStr+";"+string(acctAssetABytes)+";"+dataArr[2]
                    //调用corContractHoldB中指定的函数,"7"代表联通
                    invokeArgs:=util.ToChaincodeArgs("invoke",corContractHoldB.ContractFunctionName,contractHold.CorContractNB,"7",accIdStr,pwStr,"0",string(contractHoldBytes),allData,"mychannel",chaincodeToCall)
                    response := stub.InvokeChaincode(corContractHoldB.ContractFunctionName, invokeArgs, channelId)
                    if response.Status!=shim.OK{
                        errStr := fmt.Sprintf("getContractHold Failed to invoke chaincode. Got error: %s", string(response.Payload))
                        fmt.Printf(errStr)
                        return shim.Error(errStr)
                    }
                }
                contractHold.ContractStatus="2"
            }
        }else{
            return shim.Error("unvalid Status")
        }
    }else{
        var accIdN string
        if accIdNStr=="1"{
            accIdN=contractHold.AccIdA
        }else if accIdNStr=="2"{
            accIdN=contractHold.AccIdB
        }else if accIdNStr=="3"{
            accIdN=contractHold.AccIdC
        }else if accIdNStr=="4"{
            accIdN=contractHold.AccIdD
        }else if accIdNStr=="5"{
            accIdN=contractHold.AccIdE
        }
        if accIdStr!=accIdN{
            return shim.Error("accId is wrong")
        }
        //获得权限
        result=getPermission(stub,accIdStr,pwStr,chaincodeToCall,channelId)
        if result!=success{
            return shim.Error(result)
        }
        //获得最新的账户流水编号
        accFlowIdStr,result=getAccFlowId(stub,chaincodeToCall,channelId)
        if result!=success{
            return shim.Error(result)
        }
        accFlowId,err:=strconv.Atoi(accFlowIdStr)
        if err!=nil{
            return shim.Error("accFlowId Atoi failed")
        }
        var result string
        //获得账户A的资金账户
        acctAssetA,result=getAcctAsset(stub,contractHold.AccIdA,chaincodeToCall,channelId)
        if result!=success{
            return shim.Error(result)
        }
        //获得B的资金账户
        acctAssetB,result=getAcctAsset(stub,contractHold.AccIdB,chaincodeToCall,channelId)
        if result!=success{
            return shim.Error(result)
        }
        if contractHold.ContractStatus=="0"&&actionType=="1"{   
            //判断资金是否够锁钱
            if acctAssetA.AvaMoney<5000{
                return shim.Error("accIdA avamoney is not enough")
            }
            contractHold.ContractStatus="6"
            //A给B转钱
            acctAssetA.AvaMoney-=5000
            acctAssetB.AvaMoney+=5000
            //钱账户流水
            timeStr:=getCurrTime(stub,chaincodeToCall,channelId)
            accFlowAAsset=AccFlow{
                "accFlow",
                accFlowIdStr,
                contractHold.AccIdA,
                "money",
                "5000",
                "1",
                contractNStr,
                timeStr,
            }
            accFlowId=accFlowId+1
            accFlowIdStr=strconv.Itoa(accFlowId)
            accFlowBAsset=AccFlow{
                "accFlow",
                accFlowIdStr,
                contractHold.AccIdB,
                "money",
                "5000",
                "0",
                contractNStr,
                timeStr,
            }
            contractHold.ContractStatus="6"
            if contractHold.CorContractNB!=""{
                isTermination=false
                accFlowId=accFlowId+1
                accFlowIdStr=strconv.Itoa(accFlowId)
                corContractHoldB,result:=getContractHold(stub,contractHold.CorContractNB,chaincodeToCall,channelId)
                if result!=success{
                    return shim.Error(result)
                }
                //构造调用参数
                contractHoldBytes,err:=json.Marshal(contractHold)
                if err!=nil{
                    return shim.Error("contractHoldBytes Marshal failed")
                }
                acctAssetABytes,err:=json.Marshal(acctAssetA)
                if err!=nil{
                    return shim.Error("acctAssetABytes Marshal failed")
                }
                allData=accFlowIdStr+";"+string(acctAssetABytes)
                //调用关联合约corContractNB的持仓表里函数
                invokeArgs:=util.ToChaincodeArgs("invoke",corContractHoldB.ContractFunctionName,contractHold.CorContractNB,"7",accIdStr,pwStr,"0",string(contractHoldBytes),allData,"mychannel",chaincodeToCall)
                response := stub.InvokeChaincode(corContractHoldB.ContractFunctionName, invokeArgs, channelId)
                if response.Status!=shim.OK{
                    errStr := fmt.Sprintf("getContractHold Failed to invoke chaincode. Got error: %s", string(response.Payload))
                    fmt.Printf(errStr)
                    return shim.Error(errStr)
                }
            }
            contractHold.ContractStatus="2"
        }else if contractHold.ContractStatus=="2"&&actionType=="3"{
            if accIdStr!=contractHold.AccIdA{
                return shim.Error("accId is wrong")
            }
            timeS:=getCurrTime(stub,chaincodeToCall,channelId)
            t,err:=strconv.Atoi(timeS)
            if err!=nil{
                return shim.Error("timeS Atoi failed")
            }
            if t<20170728&&t>20170807{
                return shim.Error("time is wrong")
            }
            //获得均价
            invokeArgs:=util.ToChaincodeArgs("invoke","getAveragePrice","20170710","20170725","sh0003")
            response := stub.InvokeChaincode(chaincodeToCall, invokeArgs, channelId)
            if response.Status!=shim.OK{
                errStr := fmt.Sprintf("7 Failed to invoke chaincode. Got error: %s", string(response.Payload))
                fmt.Printf(errStr)
                return shim.Error(errStr)
            }
            //均值 float64
            averPrice,err:=strconv.ParseFloat(string(response.Payload),64) 
            if err!=nil{
                return shim.Error("averagePrice Atoi failed")
            }
            invokeArgs=util.ToChaincodeArgs("invoke","getStockPrice","20170105","sh0003")
            response = stub.InvokeChaincode(chaincodeToCall, invokeArgs, channelId)
            if response.Status!=shim.OK{
                errStr := fmt.Sprintf("7 Failed to invoke chaincode. Got error: %s", string(response.Payload))
                fmt.Printf(errStr)
                return shim.Error(errStr)
            }
            price,err:=strconv.ParseFloat(string(response.Payload),64) 
            if err!=nil{
                return shim.Error("priceBytes Atoi failed")
            }
            amt:=math.Max(averPrice-price,0)*1000
            //B给A转钱
            if acctAssetB.AvaMoney<int(amt){
                return shim.Error("acctB do not have enough money")
            }
            acctAssetB.AvaMoney-=int(amt)
            acctAssetA.AvaMoney+=int(amt)
            //钱账户流水
            timeStr:=getCurrTime(stub,chaincodeToCall,channelId)
            accFlowAAsset=AccFlow{
                "accFlow",
                accFlowIdStr,
                contractHold.AccIdA,
                "money",
                strconv.Itoa(int(amt)),//应该是amt
                "0",
                contractNStr,
                timeStr,
            }
            accFlowId=accFlowId+1
            accFlowIdStr=strconv.Itoa(accFlowId)
            accFlowBAsset=AccFlow{
                "accFlow",
                accFlowIdStr,
                contractHold.AccIdB,
                "money",
                strconv.Itoa(int(amt)),//应该是amt
                "1",
                contractNStr,
                timeStr,
            }
            contractHold.ContractStatus="4"
        }else{
            return shim.Error("unvalid Status")
        }
    }
	//保存最新的状态
    result=saveContractHold(stub,contractHold,contractNStr,chaincodeToCall,channelId)
    if result!=success{
            return shim.Error(result)
    }
    if isTermination{
        fmt.Println("lendingpro isTermination save") 
        //保存A账户资金账户和持仓账户
        result=saveAcctAsset(stub,acctAssetA,acctAssetA.AcctId,chaincodeToCall,channelId)
        if result!=success{
            return shim.Error(result)
        }
        //保存最新的账户流水号
        result=saveAccFlowId(stub,accFlowIdStr,chaincodeToCall,channelId)
        if result!=success{
            return shim.Error(result)
        }
    }
    //保存资金账户
    result=saveAcctAsset(stub,acctAssetB,acctAssetB.AcctId,chaincodeToCall,channelId)
    if result!=success{
    	return shim.Error(result)
    }
    //保存账户流水
    result=saveAccFlow(stub,accFlowAAsset,accFlowAAsset.AccFlowId,chaincodeToCall,channelId)
    if result!=success{
    	return shim.Error(result)
    }
    result=saveAccFlow(stub,accFlowBAsset,accFlowBAsset.AccFlowId,chaincodeToCall,channelId)
    if result!=success{
    	return shim.Error(result)
    }
    if allData==""{
        allData=success
    }
	return shim.Success([]byte(allData))
}
//获得系统时间
func getCurrTime(stub shim.ChaincodeStubInterface,chaincodeToCall,channelId string)string{
    invokeArgs:=util.ToChaincodeArgs("invoke","getCurrTime")
    response:= stub.InvokeChaincode(chaincodeToCall, invokeArgs, channelId)
    if response.Status!=shim.OK{
        errStr := fmt.Sprintf("getCurrTime Failed to invoke chaincode. Got error: %s", string(response.Payload))
        fmt.Printf(errStr)
        return errStr
    }
    return string(response.Payload)
}
//保存持仓账户信息
func saveSSEHoldByAccAndProduct(stub shim.ChaincodeStubInterface,sseHold,accIdStr,productCode,chaincodeToCall,channelId string)string{
    invokeArgs:=util.ToChaincodeArgs("invoke","saveSSEHoldByAccAndProduct",accIdStr,productCode,sseHold)
    response := stub.InvokeChaincode(chaincodeToCall, invokeArgs, channelId)
    if response.Status!=shim.OK{
        errStr := fmt.Sprintf("A B lockMoney Failed to invoke chaincode. Got error: %s", string(response.Payload))
        fmt.Printf(errStr)
        return errStr
    }
    return success
}
//保存最新的账户流水编号
func saveAccFlowId(stub shim.ChaincodeStubInterface,accFlowIdStr,chaincodeToCall,channelId string)string{
    invokeArgs:=util.ToChaincodeArgs("invoke","saveAccFlowId",accFlowIdStr)
    response:=stub.InvokeChaincode(chaincodeToCall, invokeArgs, channelId)
    if response.Status!=shim.OK{
        errStr := fmt.Sprintf("B saveAcctAsset Failed to invoke chaincode. Got error: %s", string(response.Payload))
        fmt.Printf(errStr)
        return errStr
    }
    return success
}
//保存合约最新的状态
func saveContractHold(stub shim.ChaincodeStubInterface,contractHold ContractHold,contractNStr,chaincodeToCall,channelId string)string{
    contractHoldBytes,err:=json.Marshal(contractHold)
    if err!=nil{
        return "contractHold marshal failed"
    }
    fmt.Println("saveContractHold"+string(contractHoldBytes))
    invokeArgs:=util.ToChaincodeArgs("invoke","saveContractHold",contractNStr,string(contractHoldBytes))
    response := stub.InvokeChaincode(chaincodeToCall, invokeArgs, channelId)
    if response.Status!=shim.OK{
        errStr := fmt.Sprintf("7 Failed to invoke chaincode. Got error: %s", string(response.Payload))
        fmt.Printf(errStr)
        return errStr
    }
    return success
}
//获得最新的账户流水编号
func getAccFlowId(stub shim.ChaincodeStubInterface,chaincodeToCall,channelId string)(string,string){
    invokeArgs:=util.ToChaincodeArgs("invoke","getAccFlowId")
    response := stub.InvokeChaincode(chaincodeToCall,invokeArgs,channelId)
    if response.Status!=shim.OK{
        errStr := fmt.Sprintf("2 getContractHold Failed to invoke chaincode. Got error: %s", string(response.Payload))
        fmt.Printf(errStr)
        return "0",errStr
    }
    return string(response.Payload),success
}
//保存账户流水
func saveAccFlow(stub shim.ChaincodeStubInterface,accFlow AccFlow,accIdStr,chaincodeToCall,channelId string)string{
    accFlowBytes,err:=json.Marshal(accFlow)
    if err!=nil{
        return "accFlowBytes Marshal failed"
    }
    invokeArgs:=util.ToChaincodeArgs("invoke","saveAccFlowOut",accIdStr,string(accFlowBytes))
    response := stub.InvokeChaincode(chaincodeToCall, invokeArgs, channelId)
    if response.Status!=shim.OK{
        errStr := fmt.Sprintf("getContractHold Failed to invoke chaincode. Got error: %s", string(response.Payload))
        fmt.Printf(errStr)
        return errStr
    }
    return success
}
//获得合约持仓
func getContractHold(stub shim.ChaincodeStubInterface,contractNStr,chaincodeToCall,channelId string)(ContractHold,string){
    var contractHold ContractHold
    invokeArgs:=util.ToChaincodeArgs("invoke","getContractHold",contractNStr)
    response := stub.InvokeChaincode(chaincodeToCall, invokeArgs, channelId)
    if response.Status!=shim.OK{
        errStr := fmt.Sprintf("getContractHold Failed to invoke chaincode. Got error: %s", string(response.Payload))
        fmt.Printf(errStr)
        return contractHold,errStr
    }
    contractHoldBytes:=response.Payload
    err:=json.Unmarshal(contractHoldBytes,&contractHold)
    if err!=nil{
        return contractHold,"contractHoldBytes Unmarshal failed"
    }
    return contractHold,success
}
//获得资金账户
func getAcctAsset(stub shim.ChaincodeStubInterface,accIdStr,chaincodeToCall,channelId string)(AcctAsset,string){
    var acctAsset AcctAsset
    invokeArgs:=util.ToChaincodeArgs("invoke","getAcctAsset",accIdStr)
    response:= stub.InvokeChaincode(chaincodeToCall, invokeArgs, channelId)
    if response.Status!=shim.OK{
        errStr := fmt.Sprintf("getContractHold Failed to invoke chaincode. Got error: %s", string(response.Payload))
        fmt.Printf(errStr)
        return acctAsset,errStr
    }
    acctAssetBytes:=response.Payload
    err:=json.Unmarshal(acctAssetBytes,&acctAsset)
    if err!=nil{
        return acctAsset,"acctAssetABytes Unmarshal failed"
    }
    return acctAsset,success
}
//保存资金账户
func saveAcctAsset(stub shim.ChaincodeStubInterface,acctAsset AcctAsset,accIdStr,chaincodeToCall,channelId string)string{
    acctAssetBytes,err:=json.Marshal(acctAsset)
    if err!=nil{
        return "acctAssetBytes Marshal failed"
    }
    invokeArgs:=util.ToChaincodeArgs("invoke","saveAcctAsset",accIdStr,string(acctAssetBytes))
    response:= stub.InvokeChaincode(chaincodeToCall, invokeArgs, channelId)
    if response.Status!=shim.OK{
        errStr := fmt.Sprintf("getContractHold Failed to invoke chaincode. Got error: %s", string(response.Payload))
        fmt.Printf(errStr)
        return errStr
    }
    return success
}
//获得权限
func getPermission(stub shim.ChaincodeStubInterface,accIdStr,pwStr,chaincodeToCall,channelId string)string{
    invokeArgs:=util.ToChaincodeArgs("invoke","getAccPermissionOut",accIdStr,pwStr)
    response := stub.InvokeChaincode(chaincodeToCall, invokeArgs, channelId)
    if response.Status!=shim.OK{
        errStr := fmt.Sprintf(" 1 getContractHold Failed to invoke chaincode. Got error: %s", string(response.Payload))
        fmt.Printf(errStr)
        return errStr+"getPermission"
    }
    return success
}
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}
