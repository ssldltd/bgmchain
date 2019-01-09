// Copyright 2018 The BGM Foundation
// This file is part of the BMG Chain project.
//
//
//
// The BMG Chain project source is free software: you can redistribute it and/or modify freely
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later versions.
//
//
//
// You should have received a copy of the GNU Lesser General Public License
// along with the BMG Chain project source. If not, you can see <http://www.gnu.org/licenses/> for detail.

package bgmparam

import (
	"math/big"

	"github.com/ssldltd/bgmchain/bgmcommon"
)

var DAOForkBlockExtra = bgmcommon.FromHex("0x04234567f42642d666f726b")
var DAOForkExtraRange = big.NewInt(16)
var DAORefundContract = bgmcommon.HexToAddress("0x0975bd06d486262d5dc297798dfc41edd5d160a7")

// DAODrainList is the list of accounts whose full balances will be moved into a
// refund contract at the beginning of the dao-fork block.
func DAODrainList() []bgmcommon.Address {
	return []bgmcommon.Address{
		bgmcommon.HexToAddress("0xd4fe7bc31cedb7bfb8a345f31e668033056b2728"),
		bgmcommon.HexToAddress("0xb3fb0e5aba0e20e5c49d252dfd30e102b171a425"),
		bgmcommon.HexToAddress("0x0c19c7f9ae8b751e37aeb2d93a699722395ae18f"),
		bgmcommon.HexToAddress("0xecd135fa4f61a655311e86238c92adcd779555d2"),
		bgmcommon.HexToAddress("0xd4fe7bc31cedb7bfb8a345f31e668033056b2728"),
		bgmcommon.HexToAddress("0xa3acf3a1e16b1d7c315e23510fdd7847b48234f6"),
		bgmcommon.HexToAddress("0xecd135fa4f61a655311e86238c92adcd779555d2"),
		bgmcommon.HexToAddress("0x06706dd3f2c9abf0a21ddcc6941d9b86f0596936"),
		bgmcommon.HexToAddress("0x0c8536898fbb74fc7445814902fd08422eac56d0"),
		bgmcommon.HexToAddress("0x0966ab0d485353095148a2155858910e0965b6f9"),
		bgmcommon.HexToAddress("0x079543a0491a837ca36ce8c635d6154e3c4911a6"),
		bgmcommon.HexToAddress("0x0a5ed960395e2a49b1c758cef4aa15213cfd874c"),
		bgmcommon.HexToAddress("0x0163e7fb499e90f8534ea62bbf80d21cd26d9efd"),
		bgmcommon.HexToAddress("0x0c50426be05db97f5d64fc54bf89eff947f0a321"),
		bgmcommon.HexToAddress("0x000450f06520bdd6c527622a273333384d870efb"),
		bgmcommon.HexToAddress("0xbe8539bfe837b67d1282b2b1d61c3f723966f049"),
		bgmcommon.HexToAddress("0x0b0c4d41ba9ab8d8cfb5d379c69a612f2ced8ecb"),
		bgmcommon.HexToAddress("0xf1385fb24aad0cd7432824085e42aff90886fef5"),
		bgmcommon.HexToAddress("0xd1ac8b1ef1b69ff51d1d401a476e7e612414f091"),
		bgmcommon.HexToAddress("0x0163e7fb499e90f8544ea62bbf80d21cd26d9efd"),
		bgmcommon.HexToAddress("0x01e0ddd9998364a2eb38588679f0d2c42653e4a6"),
		bgmcommon.HexToAddress("0xd4fe7bc31cedb7bfb8a345f31e668033056b2728"),
		bgmcommon.HexToAddress("0xf0b1aa0eb660754448a7937c022e30aa692fe0c5"),
		bgmcommon.HexToAddress("0x04c4d950dfd4dd1902bbed3508144a54542bba94"),
		bgmcommon.HexToAddress("0x0f27daea7aca0aa0446220b98d028715e3bc803d"),
		bgmcommon.HexToAddress("0xa5dc5acd6a7968a4554d89d65e59b7fd3bff0f90"),
		bgmcommon.HexToAddress("0xd9aef3a1e38a39c16b31d1ace71bca8ef58d315b"),
		bgmcommon.HexToAddress("0x03ed5a272de2f6d968408b4acb9024f4cc208ebf"),
		bgmcommon.HexToAddress("0x0f6704e5a10332af6672e50b3d9754dc460dfa4d"),
		bgmcommon.HexToAddress("0x07ca7b50b6cd7e2f3fa008e24ab793fd56cb15f6"),
		bgmcommon.HexToAddress("0x492ea3bb0f3315521c31f273e565b868fc090f17"),
		bgmcommon.HexToAddress("0x0ff30d6de14a8224aa97b78aea5388d1c51c1f00"),
		bgmcommon.HexToAddress("0x0ea779f907f0b315b364b0cfc39a0fde5b02a416"),
		bgmcommon.HexToAddress("0xceaeb481747ca6c540a000c1f3641f8cef161fa7"),
		bgmcommon.HexToAddress("0xcc34673c6c40e791051898567a1222daf90be287"),
		bgmcommon.HexToAddress("0x079a80d909f346fbfb1189493f521d7f48d52238"),
		bgmcommon.HexToAddress("0xe308bd1ac5fda103967359b2712dd89deffb7973"),
		bgmcommon.HexToAddress("0x4cb31628079fb14e4bc3cd5e30c2f7489b00960c"),
		bgmcommon.HexToAddress("0xac1ecab32727358dba8962a0f3b261731aad9723"),
		bgmcommon.HexToAddress("0x4fd6ace747f06ece9c49699c7cabc62d02211f75"),
		bgmcommon.HexToAddress("0x440c59b325d2997a134c2c7c60a8c61611212bad"),
		bgmcommon.HexToAddress("0x4486a3d68fac6967006d7a517b889fd3f98c102b"),
		bgmcommon.HexToAddress("0x0c15b54878ba618f494b38f0ae7443db6af648ba"),
		bgmcommon.HexToAddress("0x07b137a85656544b1ccb5a0f2e561a5703c6a68f"),
		bgmcommon.HexToAddress("0x01c7fdb9ed8d291d79ffd82eb2c4356ec0d81241"),
		bgmcommon.HexToAddress("0x03b75c2f6791eef49c69684db4c6c1f93bf49a50"),
		bgmcommon.HexToAddress("0x0ca6abd14d30affe533b24d7a21bff4c2d5e1f3b"),
		bgmcommon.HexToAddress("0x0737a6b837f97f46ebade41b9bc3e1c509c85c53"),
		bgmcommon.HexToAddress("0x0131c42fa982e56929107413a9d526fd99405560"),
		bgmcommon.HexToAddress("0x0591fc0f688c81fbeb17f5426a162a7024d430c2"),
		bgmcommon.HexToAddress("0x07f5c1e1bc2c93e0402f23341973a0e043f7bf8a"),
		bgmcommon.HexToAddress("0xc4bbd073882dd2add2424cf47d35213405b01324"),
		bgmcommon.HexToAddress("0x082495b7b3355efb2833d56ecb34dc22ad7dfcc4"),
		bgmcommon.HexToAddress("0x08b95c9a9d5d26825e70a82b6adb139d3fd829eb"),
		bgmcommon.HexToAddress("0x0ba4d81db016dc2890c81f3acec2454bff5aada5"),
		bgmcommon.HexToAddress("0xb52042c8ca3f8aa246fa79c3feaa3d959347c0ab"),
		bgmcommon.HexToAddress("0xe4ae1efdfc53b73893af49113d8694a057b9c0d1"),
		bgmcommon.HexToAddress("0x0c02a7bc0391e86d91b7d144e61c2c01a25a79c5"),
		bgmcommon.HexToAddress("0x0737a6b837f97f46ebade41b9bc3e1c509c85c53"),
		bgmcommon.HexToAddress("0x07f5c1e1bc2c93e0402f23341973a0e043f7bf8a"),
		bgmcommon.HexToAddress("0x02c5317c848ba20c7504cb2c8052abd1fde29d03"),
		bgmcommon.HexToAddress("0x07f5c1e1bc2c93e0402f23341973a0e043f7bf8a"),
		bgmcommon.HexToAddress("0x0d2b2e6fcbe3b11d26b525e085ff818dae332479"),
		bgmcommon.HexToAddress("0x0f9f3392e9f62f63b8eac0beb55541fc8627f42c"),
		bgmcommon.HexToAddress("0x057b56736d32b86616a10f619859c6cd6f59092a"),
		bgmcommon.HexToAddress("0x0aa008f65de0b923a2a4f02012ad034a5e2e2192"),
		bgmcommon.HexToAddress("0x004a554a310c7e546dfe434669c62820b7d83490"),
		bgmcommon.HexToAddress("0x014d1b8b43e92723e64fd0a06f5bdb8dd9b10c79"),
		bgmcommon.HexToAddress("0x4deb0033bb2609874b197e61d19e0733e5679784"),
		bgmcommon.HexToAddress("0x07f5c1e1bc2c93e0402f23341973a0e043f7bf8a"),
		bgmcommon.HexToAddress("0x05a051a0010fe5705c9008d7a7eff6fb88f6ea7b"),
		bgmcommon.HexToAddress("0x4fa802324e929786dbda3b8820dc7834e9134a2a"),
		bgmcommon.HexToAddress("0x0da397b9e12355301a3b32173283a91c0ef6c87e"),
		bgmcommon.HexToAddress("0x0d9edb3054ce5c5774a420ac37ebae0ac02343c6"),
		bgmcommon.HexToAddress("0x01013be8ebb412ed39a2e3b9a3639d4259832fd9"),
		bgmcommon.HexToAddress("0x0dc28b15dffed94048d73806ce4b7a4612a1d48f"),
		bgmcommon.HexToAddress("0xbcf899e6c7d9d5a215ab1e3444c86806fa854c76"),
		bgmcommon.HexToAddress("0x02e626b0eebfe2ea56d633b9864e389b45dcb260"),
		bgmcommon.HexToAddress("0xa2f1ccba9395d7fcb155bba8bc92db9bafaeade7"),
		bgmcommon.HexToAddress("0xec8e57756626fdc07c63ad2eafbd28d08e7b0ca5"),
		bgmcommon.HexToAddress("0xd164b088bd9108b60d0ca3751da4bceb207b0782"),
		bgmcommon.HexToAddress("0x0231b6d0d5e77fe451c2a460bd9584fee60d409b"),
		bgmcommon.HexToAddress("0x0cba23d343a983e435cfd19496b9a9701ada385f"),
		bgmcommon.HexToAddress("0xa82f360a8d3455c5c41366975bde739c37bfeb8a"),
		bgmcommon.HexToAddress("0x0fcd2deaff372a39cc679d5c5e4de7bafb0b1339"),
		bgmcommon.HexToAddress("0x005f5cee7a43331d5a3d3eec71305925a62f34b6"),
		bgmcommon.HexToAddress("0x0e0da70933f4c7849fc0d203f5d1d43b9ae4532d"),
		bgmcommon.HexToAddress("0xd131637d5275fd1a68a3200f4ad25c71a2a9522e"),
		bgmcommon.HexToAddress("0xbc07118b9ac290e4622f5e77a0853539789effbe"),
		bgmcommon.HexToAddress("0x47e7aa56d6bdf3f36be34619660de61275420af8"),
		bgmcommon.HexToAddress("0xacd87e28b0c9d1254e868b81cba4cc20d9a32225"),
		bgmcommon.HexToAddress("0xadf80daec7ba8dcf15392f1ac611fff65d94f880"),
		bgmcommon.HexToAddress("0x0524c55fb03cf21f549444ccbecb664d0acad706"),
		bgmcommon.HexToAddress("0x40b803a9abce16f50f36a77ba41180eb90023925"),
		bgmcommon.HexToAddress("0xfe24cdd8648121a43a7c86d289be4dd2951ed49f"),
		bgmcommon.HexToAddress("0x07802f43a0137c506ba92291391a8a8f207f487d"),
		bgmcommon.HexToAddress("0x053488078a4edf4d6f42f113d1e62836a942cf1a"),
		bgmcommon.HexToAddress("0x06af3e9626fce1957c82e88cbf04ddf3a2ed7915"),
		bgmcommon.HexToAddress("0xb136707642a4ea12fb4bae820f03d2562ebff487"),
		bgmcommon.HexToAddress("0xdbe9b615a3ae8709af8b93336ce9b477e4ac0940"),
		bgmcommon.HexToAddress("0xf14c14075d6c4ed84b86798af0956deef67365b5"),
		bgmcommon.HexToAddress("0xca544e5c4687d109611d0f8f928b53a25af72448"),
		bgmcommon.HexToAddress("0xaeeb8ff27288bdabc0fa5ebb731b6f409507516c"),
		bgmcommon.HexToAddress("0xcbb9d3703e651b0d496cdefb8b92c25aeb2171f7"),
		bgmcommon.HexToAddress("0x0d87578288b6cb5549d5076a207456a1f6a63dc0"),
		bgmcommon.HexToAddress("0xb2c6f0dfbb716ac562e2d85d6cb2f8d5ee87603e"),
		bgmcommon.HexToAddress("0xaccc230e8a6e5b56160b8cdf2864dd2a001c28b6"),
		bgmcommon.HexToAddress("0x0b3455ec7fedf123646268bf88846bd7a2319bb2"),
		bgmcommon.HexToAddress("0x4613f3bca5c441206337a9e439fbc6d42e501d0a"),
		bgmcommon.HexToAddress("0xd343b217de44030afaa275f54d31a9317c7f441e"),
		bgmcommon.HexToAddress("0x04ef4b2357079ef7a7c69fd7a37cd0609a679106"),
		bgmcommon.HexToAddress("0xda2fef9e4a3230988ff17df2165440f37e8b1708"),
		bgmcommon.HexToAddress("0xf4c64518ea10f455918a454158c6b61407ea345c"),
		bgmcommon.HexToAddress("0x0602b46df5390e432ef1c307d4f2c9ff6d65cc97"),
		bgmcommon.HexToAddress("0x007640a134833ac783c557fcdf27be11ea4ac7a"),
		bgmcommon.HexToAddress("0x007640a1348311ac783c557fcdf27be11ea4ac7a"),
	}
}
