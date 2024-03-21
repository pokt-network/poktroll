package keeper_test

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
)

const minExecutionPeriod = 5 * time.Second

type TestSuite struct {
	suite.Suite

	sdkCtx sdk.Context
	ctx    context.Context
	addrs  []sdk.AccAddress
	// groupID         uint64
	// groupPolicyAddr sdk.AccAddress
	// policy          group.DecisionPolicy
	// groupKeeper     keeper.Keeper
	// blockTime       time.Time
	// bankKeeper      *grouptestutil.MockBankKeeper
	// accountKeeper   *grouptestutil.MockAccountKeeper
	// environment appmodule.Environment
}

func (s *TestSuite) SetupTest() {
	// s.blockTime = time.Now().Round(0).UTC()
	// key := storetypes.NewKVStoreKey(group.StoreKey)

	// testCtx := testutil.DefaultContextWithDB(s.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	// encCfg := moduletestutil.MakeTestEncodingConfig(codectestutil.CodecOptions{}, module.AppModule{}, bank.AppModule{})
	// s.addrs = simtestutil.CreateIncrementalAccounts(6)

	// // setup gomock and initialize some globally expected executions
	// ctrl := gomock.NewController(s.T())
	// s.accountKeeper = grouptestutil.NewMockAccountKeeper(ctrl)
	// for i := range s.addrs {
	// 	s.accountKeeper.EXPECT().GetAccount(gomock.Any(), s.addrs[i]).Return(authtypes.NewBaseAccountWithAddress(s.addrs[i])).AnyTimes()
	// }
	// s.accountKeeper.EXPECT().AddressCodec().Return(address.NewBech32Codec("cosmos")).AnyTimes()

	// s.bankKeeper = grouptestutil.NewMockBankKeeper(ctrl)

	// bApp := baseapp.NewBaseApp(
	// 	"group",
	// 	log.NewNopLogger(),
	// 	testCtx.DB,
	// 	encCfg.TxConfig.TxDecoder(),
	// )
	// bApp.SetInterfaceRegistry(encCfg.InterfaceRegistry)
	// banktypes.RegisterMsgServer(bApp.MsgServiceRouter(), s.bankKeeper)

	// env := runtime.NewEnvironment(runtime.NewKVStoreService(key), log.NewNopLogger(), runtime.EnvWithRouterService(bApp.GRPCQueryRouter(), bApp.MsgServiceRouter()))
	// config := group.DefaultConfig()
	// s.groupKeeper = keeper.NewKeeper(env, encCfg.Codec, s.accountKeeper, config)
	// s.ctx = testCtx.Ctx.WithHeaderInfo(header.Info{Time: s.blockTime})
	// s.sdkCtx = sdk.UnwrapSDKContext(s.ctx)

	// s.environment = env

	// // Initial group, group policy and balance setup
	// members := []group.MemberRequest{
	// 	{Address: s.addrs[4].String(), Weight: "1"}, {Address: s.addrs[1].String(), Weight: "2"},
	// }

	// s.setNextAccount()

	// groupRes, err := s.groupKeeper.CreateGroup(s.ctx, &group.MsgCreateGroup{
	// 	Admin:   s.addrs[0].String(),
	// 	Members: members,
	// })
	// s.Require().NoError(err)
	// s.groupID = groupRes.GroupId

	// policy := group.NewThresholdDecisionPolicy(
	// 	"2",
	// 	time.Second,
	// 	minExecutionPeriod, // Must wait 5 seconds before executing proposal
	// )
	// policyReq := &group.MsgCreateGroupPolicy{
	// 	Admin:   s.addrs[0].String(),
	// 	GroupId: s.groupID,
	// }
	// err = policyReq.SetDecisionPolicy(policy)
	// s.Require().NoError(err)
	// s.setNextAccount()

	// groupSeq := s.groupKeeper.GetGroupSequence(s.sdkCtx)
	// s.Require().Equal(groupSeq, uint64(1))

	// policyRes, err := s.groupKeeper.CreateGroupPolicy(s.ctx, policyReq)
	// s.Require().NoError(err)

	// addrbz, err := address.NewBech32Codec("cosmos").StringToBytes(policyRes.Address)
	// s.Require().NoError(err)
	// s.policy = policy
	// s.groupPolicyAddr = addrbz

	// s.bankKeeper.EXPECT().MintCoins(s.sdkCtx, minttypes.ModuleName, sdk.Coins{sdk.NewInt64Coin("test", 100000)}).Return(nil).AnyTimes()
	// err = s.bankKeeper.MintCoins(s.sdkCtx, minttypes.ModuleName, sdk.Coins{sdk.NewInt64Coin("test", 100000)})
	// s.Require().NoError(err)
	// s.bankKeeper.EXPECT().SendCoinsFromModuleToAccount(s.sdkCtx, minttypes.ModuleName, s.groupPolicyAddr, sdk.Coins{sdk.NewInt64Coin("test", 10000)}).Return(nil).AnyTimes()
	// err = s.bankKeeper.SendCoinsFromModuleToAccount(s.sdkCtx, minttypes.ModuleName, s.groupPolicyAddr, sdk.Coins{sdk.NewInt64Coin("test", 10000)})
	// s.Require().NoError(err)
}

func (s *TestSuite) TestSettleClaims() {
	// addrs := s.addrs
	// addr1 := addrs[0]
	// addr2 := addrs[1]
	// votingPeriod := 4 * time.Minute
	// minExecutionPeriod := votingPeriod + group.DefaultConfig().MaxExecutionPeriod

	// groupMsg := &group.MsgCreateGroupWithPolicy{
	// 	Admin: addr1.String(),
	// 	Members: []group.MemberRequest{
	// 		{Address: addr1.String(), Weight: "1"},
	// 		{Address: addr2.String(), Weight: "1"},
	// 	},
	// }
	// policy := group.NewThresholdDecisionPolicy(
	// 	"1",
	// 	votingPeriod,
	// 	minExecutionPeriod,
	// )
	// s.Require().NoError(groupMsg.SetDecisionPolicy(policy))

	// s.setNextAccount()
	// groupRes, err := s.groupKeeper.CreateGroupWithPolicy(s.ctx, groupMsg)
	// s.Require().NoError(err)
	// accountAddr := groupRes.GetGroupPolicyAddress()
	// groupPolicy, err := s.accountKeeper.AddressCodec().StringToBytes(accountAddr)
	// s.Require().NoError(err)
	// s.Require().NotNil(groupPolicy)

	// proposalRes, err := s.groupKeeper.SubmitProposal(s.ctx, &group.MsgSubmitProposal{
	// 	GroupPolicyAddress: accountAddr,
	// 	Proposers:          []string{addr1.String()},
	// 	Messages:           nil,
	// })
	// s.Require().NoError(err)

	// _, err = s.groupKeeper.Vote(s.ctx, &group.MsgVote{
	// 	ProposalId: proposalRes.ProposalId,
	// 	Voter:      addr1.String(),
	// 	Option:     group.VOTE_OPTION_YES,
	// })
	// s.Require().NoError(err)

	// // move forward in time
	// ctx := s.sdkCtx.WithHeaderInfo(header.Info{Time: s.sdkCtx.HeaderInfo().Time.Add(votingPeriod + 1)})

	// result, err := s.groupKeeper.TallyResult(ctx, &group.QueryTallyResultRequest{
	// 	ProposalId: proposalRes.ProposalId,
	// })
	// s.Require().Equal("1", result.Tally.YesCount)
	// s.Require().NoError(err)

	// s.Require().NoError(s.groupKeeper.TallyProposalsAtVPEnd(ctx, s.environment))
	// s.NotPanics(func() {
	// 	err := s.groupKeeper.EndBlocker(ctx)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// })
}
