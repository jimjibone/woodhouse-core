// package: woodhouse.api
// file: value.proto

import * as jspb from "google-protobuf";

export class BoolValue extends jspb.Message {
  getValue(): boolean;
  setValue(value: boolean): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): BoolValue.AsObject;
  static toObject(includeInstance: boolean, msg: BoolValue): BoolValue.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: BoolValue, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): BoolValue;
  static deserializeBinaryFromReader(message: BoolValue, reader: jspb.BinaryReader): BoolValue;
}

export namespace BoolValue {
  export type AsObject = {
    value: boolean,
  }
}

export class NumberValue extends jspb.Message {
  getValue(): number;
  setValue(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): NumberValue.AsObject;
  static toObject(includeInstance: boolean, msg: NumberValue): NumberValue.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: NumberValue, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): NumberValue;
  static deserializeBinaryFromReader(message: NumberValue, reader: jspb.BinaryReader): NumberValue;
}

export namespace NumberValue {
  export type AsObject = {
    value: number,
  }
}

export class TextValue extends jspb.Message {
  getValue(): string;
  setValue(value: string): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): TextValue.AsObject;
  static toObject(includeInstance: boolean, msg: TextValue): TextValue.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: TextValue, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): TextValue;
  static deserializeBinaryFromReader(message: TextValue, reader: jspb.BinaryReader): TextValue;
}

export namespace TextValue {
  export type AsObject = {
    value: string,
  }
}

export class ColorValue extends jspb.Message {
  getHue(): number;
  setHue(value: number): void;

  getSat(): number;
  setSat(value: number): void;

  serializeBinary(): Uint8Array;
  toObject(includeInstance?: boolean): ColorValue.AsObject;
  static toObject(includeInstance: boolean, msg: ColorValue): ColorValue.AsObject;
  static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
  static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
  static serializeBinaryToWriter(message: ColorValue, writer: jspb.BinaryWriter): void;
  static deserializeBinary(bytes: Uint8Array): ColorValue;
  static deserializeBinaryFromReader(message: ColorValue, reader: jspb.BinaryReader): ColorValue;
}

export namespace ColorValue {
  export type AsObject = {
    hue: number,
    sat: number,
  }
}

