# Changelog

## [2.0.0](https://github.com/FBIKdot/reso-grab/compare/reso-grab-v1.1.0...reso-grab-v2.0.0) (2026-09-02)


### ⚠ BREAKING CHANGES

* 重写为 Go 实现

### Features

* **adapter/default:** 默认适配器 ([ced7929](https://github.com/FBIKdot/reso-grab/commit/ced7929b27e8197afe6c053a341e5613c89846d7))
* **adapter/fma:** 支持 FreeMusicArchive ([f22c446](https://github.com/FBIKdot/reso-grab/commit/f22c4462fa0adbe571f9b50b45dc0216cef52006))
* **adapter/incompetech:** incompetec 适配器 ([6033067](https://github.com/FBIKdot/reso-grab/commit/603306792760d2ce478d8b755e74b40c9453404d))
* **adapter/pixabay:** support pixabay ([e44f2fb](https://github.com/FBIKdot/reso-grab/commit/e44f2fbc7a61f33e1e727b24524deaa127d850bc))
* **adapter:** 抽象适配器 ([1852fbb](https://github.com/FBIKdot/reso-grab/commit/1852fbb25a6058065129c5a4f2e8303f46e414b2))
* add music ([6143374](https://github.com/FBIKdot/reso-grab/commit/61433741c8752ebc378f28770fc4c3195e9c056d))
* binary use quickjs engine ([de2be2d](https://github.com/FBIKdot/reso-grab/commit/de2be2db816a7d31eefb71ce1f2aa34a5fbf6619))
* compile ([f4b6213](https://github.com/FBIKdot/reso-grab/commit/f4b6213e21ae3655fa90f6a4df5e389f704c74a1))
* concurrent download ([bf912c3](https://github.com/FBIKdot/reso-grab/commit/bf912c391c10bd422baa7679fac1cf08e73a230a))
* db r/w tool ([2660d34](https://github.com/FBIKdot/reso-grab/commit/2660d34cb1f72d99bdeefb667bf9a33c44f5e117))
* **db:** addDova id exist check ([b7964a4](https://github.com/FBIKdot/reso-grab/commit/b7964a431898d7ad2d4e8ec011ffad5277c42055))
* **dm:** 专用下载管理 ([1f7e460](https://github.com/FBIKdot/reso-grab/commit/1f7e460806ad7f46619f1ad4a77995c421fbb3b9))
* download pixabay and incompetech music ([f6e3072](https://github.com/FBIKdot/reso-grab/commit/f6e3072b67cd24495c38556303e23a721f00cf8f))
* each track can set loop ([1a60baf](https://github.com/FBIKdot/reso-grab/commit/1a60baf168ecd8bf99fd7dea3f45f9de474ec4df))
* ensure db file exist ([1278546](https://github.com/FBIKdot/reso-grab/commit/127854607d5def26460bd38412bb9762ed62cd33))
* fetch优先使用br encoding ([a3fb814](https://github.com/FBIKdot/reso-grab/commit/a3fb814381283a9bf669c8811a0f8cddb3eee1f8))
* format db ([71ff50d](https://github.com/FBIKdot/reso-grab/commit/71ff50d1c86768f04a694517675c6d8644bb1181))
* loop add ([a9052bb](https://github.com/FBIKdot/reso-grab/commit/a9052bb6b8fc8da1b6aa6ed4a7753d7b7ddeb459))
* music sync (download) ([a90f332](https://github.com/FBIKdot/reso-grab/commit/a90f332cf31b8728375df556cf105763abbf18d5))
* support FreeMusicArchive.org ([ae0a70d](https://github.com/FBIKdot/reso-grab/commit/ae0a70d389437e643d4d1cbe788dd61983b9736f))
* support loop attribute ([cc64135](https://github.com/FBIKdot/reso-grab/commit/cc64135a7ddda2db4b8b8d673b2309de4e89a27b))
* version ([474c063](https://github.com/FBIKdot/reso-grab/commit/474c063b2bd75ad69ebcfd107beb3b11465b8054))
* 自定义输入 keys 顺序 ([b5d217a](https://github.com/FBIKdot/reso-grab/commit/b5d217abb88d28f64be359d6771620431aa9740e))
* 重写为 Go 实现 ([eb3b8e2](https://github.com/FBIKdot/reso-grab/commit/eb3b8e2585463265f8c09e0af1c3369e5dc79717))


### Bug Fixes

* ab在真的保存时才打印保存提示 ([5f76c46](https://github.com/FBIKdot/reso-grab/commit/5f76c46878acd60bfb8550a1a466e25aa884004d))
* **adapter/default:** 优先获取db，用于给其他适配器继承 ([6c970e3](https://github.com/FBIKdot/reso-grab/commit/6c970e383f108dcda3b58f73829751831524f46f))
* disable dova (outdated) ([5ec175e](https://github.com/FBIKdot/reso-grab/commit/5ec175e1b437e5d1f33c021bc20a80a88e2c49ce))
* handle if author and id undefined ([06acb86](https://github.com/FBIKdot/reso-grab/commit/06acb862bdeb5cc66824bf10ccd1391bf97e4fd4))
* needless `'` ([bb9d3e9](https://github.com/FBIKdot/reso-grab/commit/bb9d3e972fdb510e7b0f7d955558dc2cbcd7d90c))
* sync ensurefile ([f92ec56](https://github.com/FBIKdot/reso-grab/commit/f92ec569bcb2cb36b140a08fb7124c7394f3f769))
* wrong save path ([fd2ab6b](https://github.com/FBIKdot/reso-grab/commit/fd2ab6b69ab73ade2ef5925e290cdcf0b7a9e1f5))
* 修复数据结构 ([d7d6b03](https://github.com/FBIKdot/reso-grab/commit/d7d6b0344eeec9c364cec955069eab0d8943fd34))
* 微调id重复警告 ([f8d69dd](https://github.com/FBIKdot/reso-grab/commit/f8d69ddac60b42032be8511c18f98d0e4f54d056))
* 用import导入json而不是用fs读取 ([86cd0c7](https://github.com/FBIKdot/reso-grab/commit/86cd0c7210d7c6646df15db40ae6da66afdbe8fe))
* 调整 yaml 序列化选项 ([12df467](https://github.com/FBIKdot/reso-grab/commit/12df46770b13239a300cfef90d0871551ade7fbf))
