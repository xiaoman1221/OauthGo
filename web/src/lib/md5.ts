// 精简的 MD5 实现（Gravatar 头像 hash 使用）
export function md5(input: string): string {
  function rotateLeft(value: number, amount: number) {
    return ((value << amount) | (value >>> (32 - amount))) >>> 0
  }
  function addUnsigned(a: number, b: number) {
    const mask = 0xffffffff
    return (a + b) & mask
  }
  function toHex(value: number) {
    const hex = '0123456789abcdef'
    let output = ''
    for (let i = 7; i >= 0; i--) {
      output += hex.charAt((value >>> (i * 4)) & 0x0f)
    }
    return output
  }

  const utf8 = unescape(encodeURIComponent(input))
  let message = utf8
  let messageLength = message.length

  const blockLength = ((messageLength + 8) >> 6) + 1
  const blocks = new Array(blockLength * 16).fill(0)
  for (let i = 0; i < messageLength; i++) {
    blocks[i >> 2] |= (message.charCodeAt(i) & 0xff) << ((i % 4) * 8)
  }
  blocks[messageLength >> 2] |= 0x80 << ((messageLength % 4) * 8)
  blocks[blockLength * 16 - 2] = (messageLength * 8) & 0xffffffff
  blocks[blockLength * 16 - 1] = ((messageLength * 8) / 0x100000000) & 0xffffffff

  const s = [
    7, 12, 17, 22, 7, 12, 17, 22, 7, 12, 17, 22, 7, 12, 17, 22,
    5, 9, 14, 20, 5, 9, 14, 20, 5, 9, 14, 20, 5, 9, 14, 20,
    4, 11, 16, 23, 4, 11, 16, 23, 4, 11, 16, 23, 4, 11, 16, 23,
    6, 10, 15, 21, 6, 10, 15, 21, 6, 10, 15, 21, 6, 10, 15, 21
  ]
  const k = new Array(64)
  for (let i = 0; i < 64; i++) {
    k[i] = Math.floor(Math.abs(Math.sin(i + 1)) * 0x100000000) & 0xffffffff
  }

  let a0 = 0x67452301
  let b0 = 0xefcdab89
  let c0 = 0x98badcfe
  let d0 = 0x10325476

  for (let offset = 0; offset < blocks.length; offset += 16) {
    const m = blocks.slice(offset, offset + 16)
    let a = a0
    let b = b0
    let c = c0
    let d = d0

    for (let i = 0; i < 64; i++) {
      let f: number
      let g: number
      if (i < 16) {
        f = (b & c) | (~b & d)
        g = i
      } else if (i < 32) {
        f = (d & b) | (~d & c)
        g = (5 * i + 1) % 16
      } else if (i < 48) {
        f = b ^ c ^ d
        g = (3 * i + 5) % 16
      } else {
        f = c ^ (b | ~d)
        g = (7 * i) % 16
      }
      const temp = addUnsigned(addUnsigned(addUnsigned(addUnsigned(a, f), k[i]), m[g]), s[i])
      const rotated = rotateLeft(temp, s[i])
      a = d
      d = c
      c = b
      b = addUnsigned(b, rotated)
    }

    a0 = addUnsigned(a0, a)
    b0 = addUnsigned(b0, b)
    c0 = addUnsigned(c0, c)
    d0 = addUnsigned(d0, d)
  }

  return toHex(a0) + toHex(b0) + toHex(c0) + toHex(d0)
}
