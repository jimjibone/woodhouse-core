// package: woodhouse.api
// file: device.proto

import * as jspb from "google-protobuf";
import * as value_pb from "./value_pb";

export class DeviceInfo extends jspb.Message {
  getBridgeId(): string;
  setBridgeId(value: string): void;

  getDeviceId(): string;
  setDeviceId(value: string): void;

  getName(): string;
  setName(value: string): void;

  getDescription(): string;
  setDescription(value: string): void;

  getUrl(): string;
  setUrl(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeviceInfo.AsObject;
  static toObject(includeInstance: boolean, msg: DeviceInfo): DeviceInfo.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeviceInfo, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeviceInfo;
  static deserializeBinaryFromReader(message: DeviceInfo, reader: jspb.BinaryReader): DeviceInfo;
}

export namespace DeviceInfo {
  export type AsObject = {
    bridgeId: string,
    deviceId: string,
    name: string,
    description: string,
    url: string,
  }
}

export class DeviceState extends jspb.Message {
  getBridgeId(): string;
  setBridgeId(value: string): void;

  getDeviceId(): string;
  setDeviceId(value: string): void;

  getFullUpdate(): boolean;
  setFullUpdate(value: boolean): void;

  clearValuesList(): void;
  getValuesList(): Array<DeviceValue>;
  setValuesList(value: Array<DeviceValue>): void;
  addValues(value?: DeviceValue, index?: number): DeviceValue;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeviceState.AsObject;
  static toObject(includeInstance: boolean, msg: DeviceState): DeviceState.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeviceState, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeviceState;
  static deserializeBinaryFromReader(message: DeviceState, reader: jspb.BinaryReader): DeviceState;
}

export namespace DeviceState {
  export type AsObject = {
    bridgeId: string,
    deviceId: string,
    fullUpdate: boolean,
    valuesList: Array<DeviceValue.AsObject>,
  }
}

export class DeviceRequest extends jspb.Message {
  getBridgeId(): string;
  setBridgeId(value: string): void;

  getDeviceId(): string;
  setDeviceId(value: string): void;

  getRequestId(): string;
  setRequestId(value: string): void;

  clearValuesList(): void;
  getValuesList(): Array<DeviceValue>;
  setValuesList(value: Array<DeviceValue>): void;
  addValues(value?: DeviceValue, index?: number): DeviceValue;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeviceRequest.AsObject;
  static toObject(includeInstance: boolean, msg: DeviceRequest): DeviceRequest.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeviceRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeviceRequest;
  static deserializeBinaryFromReader(message: DeviceRequest, reader: jspb.BinaryReader): DeviceRequest;
}

export namespace DeviceRequest {
  export type AsObject = {
    bridgeId: string,
    deviceId: string,
    requestId: string,
    valuesList: Array<DeviceValue.AsObject>,
  }
}

export class DeviceResponse extends jspb.Message {
  getBridgeId(): string;
  setBridgeId(value: string): void;

  getDeviceId(): string;
  setDeviceId(value: string): void;

  getRequestId(): string;
  setRequestId(value: string): void;

  getErrorMessage(): string;
  setErrorMessage(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeviceResponse.AsObject;
  static toObject(includeInstance: boolean, msg: DeviceResponse): DeviceResponse.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeviceResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeviceResponse;
  static deserializeBinaryFromReader(message: DeviceResponse, reader: jspb.BinaryReader): DeviceResponse;
}

export namespace DeviceResponse {
  export type AsObject = {
    bridgeId: string,
    deviceId: string,
    requestId: string,
    errorMessage: string,
  }
}

export class DeviceValue extends jspb.Message {
  getName(): string;
  setName(value: string): void;

  hasBool(): boolean;
  clearBool(): void;
  getBool(): value_pb.BoolValue | undefined;
  setBool(value?: value_pb.BoolValue): void;

  hasNumber(): boolean;
  clearNumber(): void;
  getNumber(): value_pb.NumberValue | undefined;
  setNumber(value?: value_pb.NumberValue): void;

  hasText(): boolean;
  clearText(): void;
  getText(): value_pb.TextValue | undefined;
  setText(value?: value_pb.TextValue): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): DeviceValue.AsObject;
  static toObject(includeInstance: boolean, msg: DeviceValue): DeviceValue.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: DeviceValue, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): DeviceValue;
  static deserializeBinaryFromReader(message: DeviceValue, reader: jspb.BinaryReader): DeviceValue;
}

export namespace DeviceValue {
  export type AsObject = {
    name: string,
    bool?: value_pb.BoolValue.AsObject,
    number?: value_pb.NumberValue.AsObject,
    text?: value_pb.TextValue.AsObject,
  }
}

