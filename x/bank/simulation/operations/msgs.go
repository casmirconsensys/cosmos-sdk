package operations

import (
	"math/rand"

	"github.com/tendermint/tendermint/crypto"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank/internal/keeper"
	"github.com/cosmos/cosmos-sdk/x/bank/internal/types"
	"github.com/cosmos/cosmos-sdk/x/mock"
	"github.com/cosmos/cosmos-sdk/x/simulation"
)

// SimulateMsgSend tests and runs a single msg send where both
// accounts already exist.
func SimulateMsgSend(ak types.AccountKeeper, bk keeper.Keeper) simulation.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simulation.Account) (
		opMsg simulation.OperationMsg, fOps []simulation.FutureOperation, err error) {

		fromAcc, comment, msg, ok := createMsgSend(r, ctx, accs, ak)
		opMsg = simulation.NewOperationMsg(msg, ok, comment)
		if !ok {
			return opMsg, nil, nil
		}


		err = sendAndVerifyMsgSend(app, ak, msg, ctx, []crypto.PrivKey{fromAcc.PrivKey}, handler)
		if err != nil {
			return opMsg, nil, err
		}
		return opMsg, nil, nil
	}
}

func createMsgSend(r *rand.Rand, ctx sdk.Context, accs []simulation.Account, ak types.AccountKeeper) (
	fromAcc simulation.Account, comment string, msg types.MsgSend, ok bool) {

	fromAcc = simulation.RandomAcc(r, accs)
	toAcc := simulation.RandomAcc(r, accs)
	// Disallow sending money to yourself
	for {
		if !fromAcc.PubKey.Equals(toAcc.PubKey) {
			break
		}
		toAcc = simulation.RandomAcc(r, accs)
	}
	initFromCoins := ak.GetAccount(ctx, fromAcc.Address).SpendableCoins(ctx.BlockHeader().Time)

	if len(initFromCoins) == 0 {
		return fromAcc, "skipping, no coins at all", msg, false
	}

	denomIndex := r.Intn(len(initFromCoins))
	amt, err := simulation.RandPositiveInt(r, initFromCoins[denomIndex].Amount)
	if err != nil {
		return fromAcc, "skipping bank send due to account having no coins of denomination " + initFromCoins[denomIndex].Denom, msg, false
	}

	coins := sdk.Coins{sdk.NewCoin(initFromCoins[denomIndex].Denom, amt)}
	msg = types.NewMsgSend(fromAcc.Address, toAcc.Address, coins)
	return fromAcc, "", msg, true
}

// Sends and verifies the transition of a msg send.
func sendAndVerifyMsgSend(app *baseapp.BaseApp, ak types.AccountKeeper, msg types.MsgSend, ctx sdk.Context, privkeys []crypto.PrivKey, handler sdk.Handler) error {
	fromAcc := ak.GetAccount(ctx, msg.FromAddress)
	AccountNumbers := []uint64{fromAcc.GetAccountNumber()}
	SequenceNumbers := []uint64{fromAcc.GetSequence()}

	tx := mock.GenTx([]sdk.Msg{msg},
		AccountNumbers,
		SequenceNumbers,
		privkeys...)

	res := app.Deliver(tx)
	if !res.IsOK() {
		return simulation.NoOpMsg(types.ModuleName), nil, errors.New(res.Log)
	}

	return nil
}

// SimulateSingleInputMsgMultiSend tests and runs a single msg multisend, with one input and one output, where both
// accounts already exist.
func SimulateSingleInputMsgMultiSend(ak types.AccountKeeper, bk keeper.Keeper) simulation.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simulation.Account) (
		opMsg simulation.OperationMsg, fOps []simulation.FutureOperation, err error) {

		fromAcc, comment, msg, ok := createSingleInputMsgMultiSend(r, ctx, accs, ak)
		opMsg = simulation.NewOperationMsg(msg, ok, comment)
		if !ok {
			return opMsg, nil, nil
		}
		err = sendAndVerifyMsgMultiSend(app, ak, msg, ctx, []crypto.PrivKey{fromAcc.PrivKey}, handler)
		if err != nil {
			return opMsg, nil, err
		}
		return opMsg, nil, nil
	}
}

func createSingleInputMsgMultiSend(r *rand.Rand, ctx sdk.Context, accs []simulation.Account, ak types.AccountKeeper) (
	fromAcc simulation.Account, comment string, msg types.MsgMultiSend, ok bool) {

	fromAcc = simulation.RandomAcc(r, accs)
	toAcc := simulation.RandomAcc(r, accs)
	
	// Disallow sending money to yourself
	for {
		if !fromAcc.PubKey.Equals(toAcc.PubKey) {
			break
		}
		toAcc = simulation.RandomAcc(r, accs)
	}

	toAddr := toAcc.Address
	initFromCoins := ak.GetAccount(ctx, fromAcc.Address).SpendableCoins(ctx.BlockHeader().Time)

	if len(initFromCoins) == 0 {
		return fromAcc, "skipping, no coins at all", msg, false
	}

	denomIndex := r.Intn(len(initFromCoins))
	amt, err := simulation.RandPositiveInt(r, initFromCoins[denomIndex].Amount)
	if err != nil {
		return fromAcc, "skipping bank send due to account having no coins of denomination " + initFromCoins[denomIndex].Denom, msg, false
	}

	coins := sdk.Coins{sdk.NewCoin(initFromCoins[denomIndex].Denom, amt)}
	msg = types.MsgMultiSend{
		Inputs:  []types.Input{types.NewInput(fromAcc.Address, coins)},
		Outputs: []types.Output{types.NewOutput(toAddr, coins)},
	}

	return fromAcc, "", msg, true
}

// Sends and verifies the transition of a msg multisend. This fails if there are repeated inputs or outputs
// pass in handler as nil to handle txs, otherwise handle msgs
func sendAndVerifyMsgMultiSend(app *baseapp.BaseApp, ak types.AccountKeeper, msg types.MsgMultiSend,
	ctx sdk.Context, privkeys []crypto.PrivKey, handler sdk.Handler) error {

	initialInputAddrCoins := make([]sdk.Coins, len(msg.Inputs))
	initialOutputAddrCoins := make([]sdk.Coins, len(msg.Outputs))
	AccountNumbers := make([]uint64, len(msg.Inputs))
	SequenceNumbers := make([]uint64, len(msg.Inputs))

	for i := 0; i < len(msg.Inputs); i++ {
		acc := ak.GetAccount(ctx, msg.Inputs[i].Address)
		AccountNumbers[i] = acc.GetAccountNumber()
		SequenceNumbers[i] = acc.GetSequence()
		initialInputAddrCoins[i] = acc.GetCoins()
	}

	for i := 0; i < len(msg.Outputs); i++ {
		acc := ak.GetAccount(ctx, msg.Outputs[i].Address)
		initialOutputAddrCoins[i] = acc.GetCoins()
	}
	
	tx := mock.GenTx([]sdk.Msg{msg},
		AccountNumbers,
		SequenceNumbers,
		privkeys...)

	res := app.Deliver(tx)
	if !res.IsOK() {
		return simulation.NoOpMsg(types.ModuleName), nil, errors.New(res.Log)
	}

	return nil
}