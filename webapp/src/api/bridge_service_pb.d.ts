import * as jspb from 'google-protobuf'

import * as bridge_pb from './bridge_pb';
import * as device_pb from './device_pb';


export class SetBridgeInfoResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SetBridgeInfoResponse.AsObject;
  static toObject(includeInstance: boolean, msg: SetBridgeInfoResponse): SetBridgeInfoResponse.AsObject;
  static serializeBinaryToWriter(message: SetBridgeInfoResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SetBridgeInfoResponse;
  static deserializeBinaryFromReader(message: SetBridgeInfoResponse, reader: jspb.BinaryReader): SetBridgeInfoResponse;
}

export namespace SetBridgeInfoResponse {
  export type AsObject = {
  }
}

export class SetDeviceInfoResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SetDeviceInfoResponse.AsObject;
  static toObject(includeInstance: boolean, msg: SetDeviceInfoResponse): SetDeviceInfoResponse.AsObject;
  static serializeBinaryToWriter(message: SetDeviceInfoResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SetDeviceInfoResponse;
  static deserializeBinaryFromReader(message: SetDeviceInfoResponse, reader: jspb.BinaryReader): SetDeviceInfoResponse;
}

export namespace SetDeviceInfoResponse {
  export type AsObject = {
  }
}

export class SetDeviceStateResponse extends jspb.Message {
  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): SetDeviceStateResponse.AsObject;
  static toObject(includeInstance: boolean, msg: SetDeviceStateResponse): SetDeviceStateResponse.AsObject;
  static serializeBinaryToWriter(message: SetDeviceStateResponse, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): SetDeviceStateResponse;
  static deserializeBinaryFromReader(message: SetDeviceStateResponse, reader: jspb.BinaryReader): SetDeviceStateResponse;
}

export namespace SetDeviceStateResponse {
  export type AsObject = {
  }
}

export class GetDeviceRequestsRequest extends jspb.Message {
  getBridgeId(): string;
  setBridgeId(value: string): GetDeviceRequestsRequest;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): GetDeviceRequestsRequest.AsObject;
  static toObject(includeInstance: boolean, msg: GetDeviceRequestsRequest): GetDeviceRequestsRequest.AsObject;
  static serializeBinaryToWriter(message: GetDeviceRequestsRequest, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): GetDeviceRequestsRequest;
  static deserializeBinaryFromReader(message: GetDeviceRequestsRequest, reader: jspb.BinaryReader): GetDeviceRequestsRequest;
}

export namespace GetDeviceRequestsRequest {
  export type AsObject = {
    bridgeId: string,
  }
}

