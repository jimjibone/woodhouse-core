import * as jspb from 'google-protobuf'

import * as device_pb from './device_pb';


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

