// package: woodhouse.api
// file: device_store.proto

import * as jspb from "google-protobuf";
import * as bridge_pb from "./bridge_pb";
import * as device_pb from "./device_pb";

export class DeviceStore extends jspb.Message {
  clearBridgeInfosList(): void;
  getBridgeInfosList(): Array<bridge_pb.BridgeInfo>;
  setBridgeInfosList(value: Array<bridge_pb.BridgeInfo>): void;
  addBridgeInfos(value?: bridge_pb.BridgeInfo, index?: number): bridge_pb.BridgeInfo;

  clearDeviceInfosList(): void;
  getDeviceInfosList(): Array<device_pb.DeviceExtendedInfo>;
  setDeviceInfosList(value: Array<device_pb.DeviceExtendedInfo>): void;
  addDeviceInfos(value?: device_pb.DeviceExtendedInfo, index?: number): device_pb.DeviceExtendedInfo;

  clearDeviceStatesList(): void;
  getDeviceStatesList(): Array<device_pb.DeviceState>;
  setDeviceStatesList(value: Array<device_pb.DeviceState>): void;
  addDeviceStates(value?: device_pb.DeviceState, index?: number): device_pb.DeviceState;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeviceStore.AsObject;
  static toObject(includeInstance: boolean, msg: DeviceStore): DeviceStore.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeviceStore, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeviceStore;
  static deserializeBinaryFromReader(message: DeviceStore, reader: jspb.BinaryReader): DeviceStore;
}

export namespace DeviceStore {
  export type AsObject = {
    bridgeInfosList: Array<bridge_pb.BridgeInfo.AsObject>,
    deviceInfosList: Array<device_pb.DeviceExtendedInfo.AsObject>,
    deviceStatesList: Array<device_pb.DeviceState.AsObject>,
  }
}

