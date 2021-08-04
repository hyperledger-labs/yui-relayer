const IBCHost = artifacts.require("IBCHost");
const IBFT2Client = artifacts.require("IBFT2Client");
const MockClient = artifacts.require("MockClient");
const IBCClient = artifacts.require("IBCClient");
const IBCConnection = artifacts.require("IBCConnection");
const IBCChannel = artifacts.require("IBCChannel");
const IBCHandler = artifacts.require("IBCHandler");
const IBCMsgs = artifacts.require("IBCMsgs");
const IBCIdentifier = artifacts.require("IBCIdentifier");
const SimpleToken = artifacts.require("SimpleToken");
const ICS20TransferBank = artifacts.require("ICS20TransferBank");
const ICS20Bank = artifacts.require("ICS20Bank");

const deployCore = async (deployer) => {
  await deployer.deploy(IBCIdentifier);
  await deployer.link(IBCIdentifier, [IBCHost, IBFT2Client, IBCHandler]);

  await deployer.deploy(IBCMsgs);
  await deployer.link(IBCMsgs, [
    IBCClient,
    IBCConnection,
    IBCChannel,
    IBCHandler,
    IBFT2Client,
  ]);

  await deployer.deploy(IBCClient);
  await deployer.link(IBCClient, [IBCHandler, IBCConnection, IBCChannel]);

  await deployer.deploy(IBCConnection);
  await deployer.link(IBCConnection, [IBCHandler, IBCChannel]);

  await deployer.deploy(IBCChannel);
  await deployer.link(IBCChannel, [IBCHandler]);

  await deployer.deploy(IBFT2Client);
  await deployer.deploy(MockClient);

  await deployer.deploy(IBCHost);
  await deployer.deploy(IBCHandler, IBCHost.address);
};

const deployApp = async (deployer) => {
  console.log("deploying app contracts");

  await deployer.deploy(SimpleToken, "simple", "simple", 1000000);
  await deployer.deploy(ICS20Bank)
  await deployer.deploy(ICS20TransferBank, IBCHost.address, IBCHandler.address, ICS20Bank.address);
};

module.exports = async function (deployer) {
  await deployCore(deployer);
  await deployApp(deployer);
};
