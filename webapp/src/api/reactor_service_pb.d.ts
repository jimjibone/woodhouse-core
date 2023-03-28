import * as jspb from 'google-protobuf'

import * as bridge_pb from './bridge_pb';
import * as device_pb from './device_pb';


export class GetBridgeInfosRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetBridgeInfosRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetBridgeInfosRequest): GetBridgeInfosRequest.AsObject;
  static serializeBinaryToWriter(message: GetBridgeInfosRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetBridgeInfosRequest;
  static deserializeBinaryFromReader(message: GetBridgeInfosRequest, reader: jspb.BinaryReader): GetBridgeInfosRequest;
}

export namespace GetBridgeInfosRequest {
  export type AsObject = {
  }
}

export class GetDeviceInfosRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetDeviceInfosRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetDeviceInfosRequest): GetDeviceInfosRequest.AsObject;
  static serializeBinaryToWriter(message: GetDeviceInfosRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetDeviceInfosRequest;
  static deserializeBinaryFromReader(message: GetDeviceInfosRequest, reader: jspb.BinaryReader): GetDeviceInfosRequest;
}

export namespace GetDeviceInfosRequest {
  export type AsObject = {
  }
}

export class GetDeviceStatesRequest extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetDeviceStatesRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetDeviceStatesRequest): GetDeviceStatesRequest.AsObject;
  static serializeBinaryToWriter(message: GetDeviceStatesRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetDeviceStatesRequest;
  static deserializeBinaryFromReader(message: GetDeviceStatesRequest, reader: jspb.BinaryReader): GetDeviceStatesRequest;
}

export namespace GetDeviceStatesRequest {
  export type AsObject = {
  }
}

export class SetDeviceHiddenRequest extends jspb.Message {
  getBridgeId(): string;
  setBridgeId(value: string): SetDeviceHiddenRequest;

  getDeviceId(): string;
  setDeviceId(value: string): SetDeviceHiddenRequest;

  getHidden(): boolean;
  setHidden(value: boolean): SetDeviceHiddenRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SetDeviceHiddenRequest.AsObject;
  static toObject(includeInstance: boolean, msg: SetDeviceHiddenRequest): SetDeviceHiddenRequest.AsObject;
  static serializeBinaryToWriter(message: SetDeviceHiddenRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SetDeviceHiddenRequest;
  static deserializeBinaryFromReader(message: SetDeviceHiddenRequest, reader: jspb.BinaryReader): SetDeviceHiddenRequest;
}

export namespace SetDeviceHiddenRequest {
  export type AsObject = {
    bridgeId: string,
    deviceId: string,
    hidden: boolean,
  }
}

export class SetDeviceHiddenResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SetDeviceHiddenResponse.AsObject;
  static toObject(includeInstance: boolean, msg: SetDeviceHiddenResponse): SetDeviceHiddenResponse.AsObject;
  static serializeBinaryToWriter(message: SetDeviceHiddenResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SetDeviceHiddenResponse;
  static deserializeBinaryFromReader(message: SetDeviceHiddenResponse, reader: jspb.BinaryReader): SetDeviceHiddenResponse;
}

export namespace SetDeviceHiddenResponse {
  export type AsObject = {
  }
}

export class SetDeviceFavouriteRequest extends jspb.Message {
  getBridgeId(): string;
  setBridgeId(value: string): SetDeviceFavouriteRequest;

  getDeviceId(): string;
  setDeviceId(value: string): SetDeviceFavouriteRequest;

  getFavourite(): boolean;
  setFavourite(value: boolean): SetDeviceFavouriteRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SetDeviceFavouriteRequest.AsObject;
  static toObject(includeInstance: boolean, msg: SetDeviceFavouriteRequest): SetDeviceFavouriteRequest.AsObject;
  static serializeBinaryToWriter(message: SetDeviceFavouriteRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SetDeviceFavouriteRequest;
  static deserializeBinaryFromReader(message: SetDeviceFavouriteRequest, reader: jspb.BinaryReader): SetDeviceFavouriteRequest;
}

export namespace SetDeviceFavouriteRequest {
  export type AsObject = {
    bridgeId: string,
    deviceId: string,
    favourite: boolean,
  }
}

export class SetDeviceFavouriteResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SetDeviceFavouriteResponse.AsObject;
  static toObject(includeInstance: boolean, msg: SetDeviceFavouriteResponse): SetDeviceFavouriteResponse.AsObject;
  static serializeBinaryToWriter(message: SetDeviceFavouriteResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SetDeviceFavouriteResponse;
  static deserializeBinaryFromReader(message: SetDeviceFavouriteResponse, reader: jspb.BinaryReader): SetDeviceFavouriteResponse;
}

export namespace SetDeviceFavouriteResponse {
  export type AsObject = {
  }
}

