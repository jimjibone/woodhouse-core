// package: woodhouse.api
// file: bridge.proto

import * as jspb from "google-protobuf";
import * as timestamp_pb from "./timestamp_pb";

export class BridgeInfo extends jspb.Message {
  getBridgeId(): string;
  setBridgeId(value: string): void;

  getName(): string;
  setName(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  hasBootTime(): boolean;
  clearBootTime(): void;
  getBootTime(): timestamp_pb.Timestamp | undefined;
  setBootTime(value?: timestamp_pb.Timestamp): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): BridgeInfo.AsObject;
  static toObject(includeInstance: boolean, msg: BridgeInfo): BridgeInfo.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: BridgeInfo, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): BridgeInfo;
  static deserializeBinaryFromReader(message: BridgeInfo, reader: jspb.BinaryReader): BridgeInfo;
}

export namespace BridgeInfo {
  export type AsObject = {
    bridgeId: string,
    name: string,
    description: string,
    bootTime?: timestamp_pb.Timestamp.AsObject,
  }
}

