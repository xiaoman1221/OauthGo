// 第三方登录渠道元数据：一句话文档（TIPS）、接入地址、接入步骤、回调配置说明等
// 同时供「登录渠道」配置弹窗与「第三方渠道接入文档」页复用

export const channelSchemas = {
  wechat: {
    name: 'wechat',
    displayName: '微信',
    category: 'social',
    idLabel: 'AppID',
    secretLabel: 'AppSecret',
    idPlaceholder: '微信开放平台 AppID',
    secretPlaceholder: '微信开放平台 AppSecret',
    tips: '在微信开放平台（open.weixin.qq.com）创建「网站应用」获取 AppID 与 AppSecret，并将回调地址配置为授权回调域。',
    registerUrl: 'https://open.weixin.qq.com',
    registerLabel: '微信开放平台',
    fields: ['AppID', 'AppSecret', '授权回调域'],
    steps: [
      '在微信开放平台注册开发者账号并完成认证。',
      '创建「网站应用」，提交应用审核。',
      '应用审核通过后获取 AppID 与 AppSecret。',
      '在应用「开发信息 → 授权回调域」填入 OauthGo 站点域名。',
      '在 OauthGo「登录渠道」中填入 AppID 与 AppSecret，开启「应用于主站登录」并保存。'
    ],
    callbackNote: '在微信开放平台「开发信息 → 授权回调域」只需填写站点域名（去掉 https:// 与路径），例如 example.com。',
    notes: ['网站应用需通过微信认证并提交审核，周期较长。', '授权回调域只填域名，不填路径。']
  },
  wechat_miniprogram: {
    name: 'wechat_miniprogram',
    displayName: '微信小程序',
    category: 'social',
    idLabel: '小程序 AppID',
    secretLabel: '小程序 AppSecret',
    idPlaceholder: '小程序 AppID',
    secretPlaceholder: '小程序 AppSecret',
    tips: '在微信公众平台注册小程序获取 AppID 与 AppSecret，前端直接传入 code 登录，无需配置回调地址。',
    registerUrl: 'https://mp.weixin.qq.com',
    registerLabel: '微信公众平台',
    fields: ['小程序 AppID', '小程序 AppSecret'],
    steps: [
      '在微信公众平台注册并认证小程序。',
      '在「开发管理 → 开发设置」获取 AppID 与 AppSecret。',
      '在 OauthGo「登录渠道」填入 AppID 与 AppSecret 并保存。',
      '小程序前端调用 wx.login 获取 code，然后 POST /api/oauth/wechat_miniprogram/login（body 传 {code}）完成登录。'
    ],
    callbackNote: '该渠道无网页跳转，前端直接传 code 登录，无需配置回调地址。',
    notes: ['AppSecret 请妥善保管，勿在小程序端明文存储。', '该渠道不展示在主站网页登录页，仅通过接口登录。']
  },
  qq: {
    name: 'qq',
    displayName: 'QQ',
    category: 'social',
    idLabel: 'APP ID',
    secretLabel: 'APP Key',
    idPlaceholder: 'QQ 互联 APP ID',
    secretPlaceholder: 'QQ 互联 APP Key',
    tips: '在 QQ 互联（connect.qq.com）创建「网站应用」获取 APP ID 与 APP Key，并将回调地址配置为应用回调域。',
    registerUrl: 'https://connect.qq.com',
    registerLabel: 'QQ 互联',
    fields: ['APP ID', 'APP Key', '回调地址'],
    steps: [
      '在 QQ 互联注册开发者账号。',
      '创建「网站应用」，填写网站信息并提交审核。',
      '审核通过后获取 APP ID 与 APP Key。',
      '在「应用管理 → 开发信息」配置「授权回调域」（站点域名）。',
      '在 OauthGo「登录渠道」填入 APP ID 与 APP Key 并保存。'
    ],
    callbackNote: '在 QQ 互联「应用管理 → 开发信息」配置授权回调域，只需填域名。',
    notes: ['网站应用需提交审核，通常数个工作日。', '回调域填域名即可，不必填完整 URL。']
  },
  weibo: {
    name: 'weibo',
    displayName: '微博',
    category: 'social',
    idLabel: 'App Key',
    secretLabel: 'App Secret',
    idPlaceholder: '微博开放平台 App Key',
    secretPlaceholder: '微博开放平台 App Secret',
    tips: '在微博开放平台（open.weibo.com）创建「网站应用」获取 App Key 与 App Secret，并将回调地址填入 OAuth2.0 回调页。',
    registerUrl: 'https://open.weibo.com',
    registerLabel: '微博开放平台',
    fields: ['App Key', 'App Secret', 'OAuth2.0 回调页'],
    steps: [
      '在微博开放平台注册开发者并完成实名认证。',
      '创建「网页应用」，获取 App Key 与 App Secret。',
      '在「高级信息 → OAuth2.0 授权设置」填入完整回调地址。',
      '提交应用审核。',
      '在 OauthGo「登录渠道」填入 App Key 与 App Secret 并保存。'
    ],
    callbackNote: '在微博开放平台「高级信息 → OAuth2.0 授权设置」填入完整回调地址（含 https://）。',
    notes: ['需实名认证，应用审核周期较长。']
  },
  gitee: {
    name: 'gitee',
    displayName: 'Gitee',
    category: 'social',
    idLabel: 'Client ID',
    secretLabel: 'Client Secret',
    idPlaceholder: 'Gitee Client ID',
    secretPlaceholder: 'Gitee Client Secret',
    tips: '在 Gitee「设置→私人令牌→OAuth 应用」创建应用获取 Client ID 与 Client Secret，并将回调地址填入应用回调地址。',
    registerUrl: 'https://gitee.com/oauth/applications',
    registerLabel: 'Gitee OAuth 应用',
    fields: ['Client ID', 'Client Secret', '应用回调地址'],
    steps: [
      '登录 Gitee，进入「设置 → 私人令牌 → OAuth 应用」。',
      '点击「创建应用」，填写应用信息与回调地址并提交。',
      '创建成功后获取 Client ID 与 Client Secret。',
      '在 OauthGo「登录渠道」填入 Client ID 与 Client Secret 并保存。'
    ],
    callbackNote: '在 Gitee OAuth 应用的「应用回调地址」填入完整回调地址。',
    notes: ['即时生效，无需人工审核，适合测试联调。']
  },
  douyin: {
    name: 'douyin',
    displayName: '抖音',
    category: 'social',
    idLabel: 'Client Key',
    secretLabel: 'Client Secret',
    idPlaceholder: '抖音开放平台 Client Key',
    secretPlaceholder: '抖音开放平台 Client Secret',
    tips: '在抖音开放平台（developer.open-douyin.com）创建「移动应用」获取 Client Key 与 Client Secret，并将回调地址配置到「回跳地址」。',
    registerUrl: 'https://developer.open-douyin.com',
    registerLabel: '抖音开放平台',
    fields: ['Client Key', 'Client Secret', '回跳地址'],
    steps: [
      '在抖音开放平台注册开发者并创建「移动应用」。',
      '在「应用设置 → 回跳地址」填入完整回调地址。',
      '获取 Client Key 与 Client Secret。',
      '在 OauthGo「登录渠道」填入 Client Key 与 Client Secret 并保存。'
    ],
    callbackNote: '在抖音开放平台「应用设置 → 回跳地址」填入完整回调地址。',
    notes: ['部分登录能力需进行资质与经营范围审核。']
  },
  baidu: {
    name: 'baidu',
    displayName: '百度',
    category: 'social',
    idLabel: 'API Key',
    secretLabel: 'Secret Key',
    idPlaceholder: '百度智能云 API Key',
    secretPlaceholder: '百度智能云 Secret Key',
    tips: '在百度智能云（console.bce.baidu.com）安全认证→OAuth 中创建应用获取 API Key 与 Secret Key，并将回调地址配置到「授权回调地址」。',
    registerUrl: 'https://console.bce.baidu.com',
    registerLabel: '百度智能云',
    fields: ['API Key', 'Secret Key', '授权回调地址'],
    steps: [
      '登录百度智能云控制台，完成实名认证。',
      '进入「安全认证 → OAuth」创建应用。',
      '配置「OAuth 授权回调地址」。',
      '获取 API Key 与 Secret Key。',
      '在 OauthGo「登录渠道」填入 API Key 与 Secret Key 并保存。'
    ],
    callbackNote: '在百度智能云「安全认证 → OAuth → 应用详情」配置授权回调地址。',
    notes: ['需要百度账号完成实名认证。']
  },
  alipay: {
    name: 'alipay',
    displayName: '支付宝',
    category: 'social',
    idLabel: 'AppID',
    secretLabel: 'AppSecret',
    idPlaceholder: '支付宝开放平台 AppID',
    secretPlaceholder: '支付宝开放平台 AppSecret',
    tips: '在支付宝开放平台（open.alipay.com）创建应用，开启「第三方应用授权」，并将回调地址配置到「授权回调地址」。',
    registerUrl: 'https://open.alipay.com',
    registerLabel: '支付宝开放平台',
    fields: ['AppID', 'AppSecret', '应用私钥（RSA2）', '授权回调地址'],
    steps: [
      '注册支付宝开放平台账号并完成企业或个人认证。',
      '创建「自研应用」，申请开通「第三方应用授权」等登录相关权限。',
      '生成 RSA2 密钥对，将「应用公钥」配置到平台，私钥留作 OauthGo 的「应用私钥」。',
      '在「开发设置 → 授权回调地址」填入完整回调地址。',
      '在 OauthGo「登录渠道」填入 AppID、AppSecret 与应用私钥并保存。'
    ],
    callbackNote: '在支付宝开放平台「开发设置 → 授权回调地址」填入完整回调地址。',
    configFields: [
      { key: 'app_private_key', label: '应用私钥', type: 'textarea', rows: 4, placeholder: 'RSA2 应用私钥（商户私钥）' }
    ],
    divider: '扩展配置',
    notes: ['应用私钥不可外泄，务必妥善保管。', '密钥算法需选择 RSA2，公钥在平台配置、私钥填入本系统。', '相关登录权限需提交审核。']
  },
  dingtalk: {
    name: 'dingtalk',
    displayName: '钉钉',
    category: 'enterprise',
    idLabel: 'Client ID',
    secretLabel: 'Client Secret',
    idPlaceholder: '钉钉开放平台 Client ID',
    secretPlaceholder: '钉钉开放平台 Client Secret',
    tips: '在钉钉开放平台（open.dingtalk.com）创建应用获取 Client ID（AppKey）与 Client Secret（AppSecret），并将回调地址填入「登录回调域名」。',
    registerUrl: 'https://open.dingtalk.com',
    registerLabel: '钉钉开放平台',
    fields: ['Client ID（AppKey）', 'Client Secret（AppSecret）', '登录回调域名'],
    steps: [
      '在钉钉开放平台注册开发者账号。',
      '创建「企业内部应用」，获取 Client ID 与 Client Secret。',
      '在应用「登录 → 回调域名」填入站点域名。',
      '在 OauthGo「登录渠道」填入 Client ID 与 Client Secret 并保存。'
    ],
    callbackNote: '在钉钉开放平台应用「登录与分享 → 回调域名」填入站点域名。',
    notes: ['企业内部应用需在组织内授权使用。']
  },
  wecom: {
    name: 'wecom',
    displayName: '企业微信',
    category: 'enterprise',
    idLabel: 'CorpID',
    secretLabel: 'Secret',
    idPlaceholder: '企业微信企业 ID',
    secretPlaceholder: '企业微信应用 Secret',
    tips: '在企业微信管理后台「应用管理→自建」创建应用，CorpID 为企业 ID，Secret 为应用密钥，并将回调地址配置到「企业微信授权回调域」。',
    registerUrl: 'https://work.weixin.qq.com',
    registerLabel: '企业微信管理后台',
    fields: ['CorpID（企业 ID）', 'Secret（应用密钥）', 'AgentId', '授权回调域'],
    steps: [
      '注册企业微信并创建企业。',
      '在「应用管理 → 应用 → 自建」创建自建应用。',
      '获取企业 ID（CorpID）、AgentId 与应用 Secret。',
      '在应用「开发者接口 → 网页授权及 JS-SDK → 可信域名」配置站点域名。',
      '在 OauthGo「登录渠道」填入 CorpID、Secret、AgentId，并选择登录类型后保存。'
    ],
    callbackNote: '在企业微信管理后台自建应用「开发者接口 → 网页授权及 JS-SDK」配置可信域名。',
    configFields: [
      { key: 'agent_id', label: 'AgentId', type: 'input', placeholder: '企业自建应用 AgentId' },
      {
        key: 'login_type',
        label: '登录类型',
        type: 'select',
        options: [
          { label: '企业自建应用（CorpApp）', value: 'CorpApp' },
          { label: '第三方应用（ServiceApp）', value: 'ServiceApp' }
        ]
      }
    ],
    divider: '扩展配置',
    notes: [
      '登录类型：CorpApp 为企业自建应用，ServiceApp 为第三方应用。',
      '第三方应用场景下 corp_id 填服务商 corpid。',
      '配置需企业微信管理员操作。'
    ]
  },
  lark: {
    name: 'lark',
    displayName: '飞书',
    category: 'enterprise',
    idLabel: 'App ID',
    secretLabel: 'App Secret',
    idPlaceholder: '飞书开放平台 App ID',
    secretPlaceholder: '飞书开放平台 App Secret',
    tips: '在飞书开放平台（open.feishu.cn）创建「企业自建应用」获取 App ID 与 App Secret，并在「安全设置→重定向 URL」中配置回调地址。',
    registerUrl: 'https://open.feishu.cn',
    registerLabel: '飞书开放平台',
    fields: ['App ID', 'App Secret', '重定向 URL'],
    steps: [
      '在飞书开放平台创建企业并进入开发者后台。',
      '创建「企业自建应用」。',
      '在「安全设置 → 重定向 URL」填入完整回调地址。',
      '发布应用版本并等待管理员审批。',
      '获取 App ID 与 App Secret，在 OauthGo「登录渠道」填入并保存。'
    ],
    callbackNote: '在飞书开放平台应用「安全设置 → 重定向 URL」填入完整回调地址。',
    notes: ['应用需发布并审批通过后方可授权登录。']
  },
  infoflow: {
    name: 'infoflow',
    displayName: '如流',
    category: 'enterprise',
    idLabel: 'AppID',
    secretLabel: 'Secret',
    idPlaceholder: '如流开放平台 AppID',
    secretPlaceholder: '如流开放平台 Secret',
    tips: '在如流开放平台（xpc.im.baidu.com）创建应用获取 AppID 与 Secret，并将回调地址配置到「回调地址」。',
    registerUrl: 'https://xpc.im.baidu.com',
    registerLabel: '如流开放平台',
    fields: ['AppID', 'Secret', '回调地址'],
    steps: [
      '在如流开放平台注册并登录。',
      '创建应用，获取 AppID 与 Secret。',
      '在应用「回调地址」填入完整回调地址。',
      '在 OauthGo「登录渠道」填入 AppID 与 Secret 并保存。'
    ],
    callbackNote: '在如流开放平台应用「回调地址」填入完整回调地址。',
    notes: ['如流为百度旗下的企业协作产品，登录需企业授权。']
  }
}

export const channelList = Object.values(channelSchemas)

export const defaultChannelSchema = {
  idLabel: 'ClientID',
  secretLabel: 'ClientSecret',
  idPlaceholder: '',
  secretPlaceholder: '',
  tips: '',
  registerUrl: '',
  registerLabel: '',
  fields: [],
  steps: [],
  callbackNote: '',
  notes: [],
  configFields: [],
  divider: ''
}

export function channelSchema(name) {
  return { ...defaultChannelSchema, ...(channelSchemas[name] || {}) }
}
