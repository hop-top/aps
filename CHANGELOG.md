# Changelog

## [0.6.0-alpha.0](https://github.com/hop-top/aps/compare/aps/v0.5.0-alpha.3...aps/v0.6.0-alpha.0) (2026-05-25)


### ⚠ BREAKING CHANGES

* **adapter:** Manifests with an env_prefix that is not a valid env-var name component (must match ^[A-Z_][A-Z0-9_]*$ after uppercase) are now rejected at LoadManifest time. Set APS_ADAPTER_DISABLE_PREFIX_VALIDATION=1 to bypass during migration.
* **cli:** drop top-level aps messenger, keep under adapter (T-0363)
* **cli:** drop `--json` flag on `aps version`, `aps profile list`, `aps action list`. Use the persistent `--format {table|json|yaml}` flag provided by kit/go/console/cli on root.

### Features

* **a2a:** wire kit/console/progress on a2a card fetch ([ccec876](https://github.com/hop-top/aps/commit/ccec8768af7cb626c331611c1235a904da1a9add))
* **a2a:** wire kit/console/progress on a2a tasks cancel ([1958e2c](https://github.com/hop-top/aps/commit/1958e2c3c0eb667be27c4d4c3a51960afb359f17))
* **a2a:** wire kit/console/progress on a2a tasks send ([350caad](https://github.com/hop-top/aps/commit/350caad50c92512822b7772890ad597c88aea32e))
* **a2a:** wire kit/console/progress on a2a tasks subscribe ([b527380](https://github.com/hop-top/aps/commit/b527380de7d79795715a986f9e7a2798a24be6ce))
* **adapter:** email adapter + exec command + profile email field ([6fd1125](https://github.com/hop-top/aps/commit/6fd11259458f28a1e0c80314b0489331ab24aeb3))
* **adapter:** email adapter scaffold + exec command (WIP) ([8315f4f](https://github.com/hop-top/aps/commit/8315f4fb199106c63e951e9ecdf0a31e6b6968ab))
* **adapter:** register scheduler type + manifest-driven env_prefix ([#85](https://github.com/hop-top/aps/issues/85)) ([70db1e6](https://github.com/hop-top/aps/commit/70db1e6396a7166732843cb49434331d3eed3a7a))
* **adapters/calendar:** scheduler family + gcalcli/gam backends (T-0581) ([#83](https://github.com/hop-top/aps/issues/83)) ([09d2e66](https://github.com/hop-top/aps/commit/09d2e661067284344d2d1654e81d786f6320f2b6))
* add ACP support, increase test coverage ([cf24e80](https://github.com/hop-top/aps/commit/cf24e800540bd336a5baddc3dc59d47aa854003e))
* add adapter YAML export/import and profile path sanitization (T-0029..T-0030) ([6eebad6](https://github.com/hop-top/aps/commit/6eebad6609be2ac80d567be73468a43766616008))
* add agent protocol support ([bb55236](https://github.com/hop-top/aps/commit/bb552366c580bf0ef9b2e17983e1e9a462ef8da8))
* add Docker-based user journey testing environment ([ca86b0f](https://github.com/hop-top/aps/commit/ca86b0f4cf62a8110ab00f81f850cfd64db7f473))
* add isolation manager and session management system ([5597d0d](https://github.com/hop-top/aps/commit/5597d0d3a48a1e3e2174aacbb924de78f7a44a60))
* add lychee link checker for documentation ([541a477](https://github.com/hop-top/aps/commit/541a477436145c33ce4e691a09eb91b393724b6d))
* add profile isolation (macos, linux) ([9221f29](https://github.com/hop-top/aps/commit/9221f29a3a48be8e2db304e7e7d396839c0482b8))
* add shareable profile bundles ([1a17ebc](https://github.com/hop-top/aps/commit/1a17ebc9f13d65e9e5c5aeb722051ea412691746))
* add support for custom profile env var prefix (closes [#1](https://github.com/hop-top/aps/issues/1)) ([3870ce7](https://github.com/hop-top/aps/commit/3870ce74b9da21b9a5408ad06a612cff6c32d687))
* add test runner for user story-linked tests ([9b3c69e](https://github.com/hop-top/aps/commit/9b3c69e0ed0c12cec8009e5a035240b368c7f3d2))
* adopt hop.top/cxr for execution and adapter dispatch ([52140f2](https://github.com/hop-top/aps/commit/52140f235f8a4554abba61465fe946ccde512b22))
* adopt hop.top/cxr for execution and adapter dispatch ([3b83ed1](https://github.com/hop-top/aps/commit/3b83ed168d13ee23400e356525af4e93695845da))
* **agntcy:** implement AGNTCY integration (phases 1-5) ([a97430e](https://github.com/hop-top/aps/commit/a97430e7b96a01c03fd6fa11996d30f8ee9905f9))
* **audit:** wire WorkspaceAuditLog as bus subscriber ([7dc0027](https://github.com/hop-top/aps/commit/7dc0027e21adca503f9b409094e49a5f879c8b98))
* bump kit/uri/xrr/cxr + adopt release-please ([#41](https://github.com/hop-top/aps/issues/41)) ([1a9906b](https://github.com/hop-top/aps/commit/1a9906b1e5f41f4f98d6ae80e42c2943f04809bb))
* **bundle:** add built-in bundle YAML definitions (T-0057..T-0062) ([8937477](https://github.com/hop-top/aps/commit/8937477bd312e2faf99f6add17134137b918edc6))
* **bundle:** define types, loader, and registry (T-0040..T-0043) ([ca157a3](https://github.com/hop-top/aps/commit/ca157a3132065f6ea38100ce4abea67cd67c222a))
* **bundle:** implement resolver — inheritance, scope union, binary eval, env injection (T-0044..T-0051) ([b905580](https://github.com/hop-top/aps/commit/b905580a4cae81283f662ac7c8eeb4bd517c5dcd))
* **bundle:** wire bundle resolution into profile start, status, and scope (T-0052..T-0056) ([4babdd3](https://github.com/hop-top/aps/commit/4babdd373c82b943105daba7547ec945b511dfa6))
* **bus:** connect to dpkms cross-process bus hub ([a05284a](https://github.com/hop-top/aps/commit/a05284a8138f99efaf24dcdd82d75df3282bcbe9))
* **cli/a2a:** adopt listing helper for tasks list (T-0438) ([2922164](https://github.com/hop-top/aps/commit/2922164e36d8a228276a5614a060e07a08b36361))
* **cli/action:** adopt listing helper + --type filter (T-0439) ([2390dd0](https://github.com/hop-top/aps/commit/2390dd0cd68b5d4075385a777ebebc5c4f55a9b0))
* **cli/bundle:** rich row + --tag/--builtin/--user filters (T-0432) ([c629332](https://github.com/hop-top/aps/commit/c629332eefa7e5f1c3041dcf0e19d5b6a8b38000))
* **cli/capability:** rich row + 4 filters incl. patterns sibling (T-0433) ([f74cd60](https://github.com/hop-top/aps/commit/f74cd6095af3767b438d980eeb21c8c0e28f3bef))
* **cli/contact:** rich row + --org/--has-email filters (T-0434) ([6cb3bdf](https://github.com/hop-top/aps/commit/6cb3bdff26516dc180480676198a804c5be3db55))
* **cli/globals:** sweep --offline across network-touching commands (T-0411) ([37391c2](https://github.com/hop-top/aps/commit/37391c2ae99b4431f58a7da72705944a1c946dbc))
* **cli/listing:** shared helper for kit/output.Render + filter predicates (T-0427) ([52eaa38](https://github.com/hop-top/aps/commit/52eaa382ce4302edbe29f160b6e74edeece44821))
* **cli/profile:** rich list row + 7 filter flags (T-0428) ([f5aac55](https://github.com/hop-top/aps/commit/f5aac557bc819ccf0a55a7b5d45c493403cda8c4))
* **cli/run:** add --env KEY=VAL and --env-file flags with precedence ([#71](https://github.com/hop-top/aps/issues/71)) ([e454f9a](https://github.com/hop-top/aps/commit/e454f9ab31e2d703b6f2f0c757498eedcbace2bc))
* **cli/session:** rich row + 4 filter flags (T-0429) ([0caa989](https://github.com/hop-top/aps/commit/0caa98930eda72b22386f149092cf1909b61b1bf))
* **cli/skill:** adopt listing helper + flag audit (T-0440) ([74cd847](https://github.com/hop-top/aps/commit/74cd847c639c254df46cd2556617b171ddaed55d))
* **cli/squad:** rich list row + 2 filter flags (T-0431) ([cf1b1f2](https://github.com/hop-top/aps/commit/cf1b1f2b802dcc8c5b23d4ba81cec51248d5cf72))
* **cli/workspace:** rich list row + 3 filter flags (T-0430) ([9be1773](https://github.com/hop-top/aps/commit/9be1773e3933776c4c098fc9619294c461dd7364))
* **cli/workspace:** rich list rows + filters for conflicts/ctx/policy ([bd48ebe](https://github.com/hop-top/aps/commit/bd48ebeafa90f0c1acb96f2ca3f734b8b40c50f4))
* **cli:** add --note|-n to state-changing subcommands (T-1291) ([aa6f131](https://github.com/hop-top/aps/commit/aa6f13124be34c51920442af6e274adbc41c2808))
* **cli:** add --type filter to 'aps session list' (T-0364) ([f672013](https://github.com/hop-top/aps/commit/f672013ebdf46f5e1481f7d6f858f8d5b721ee27))
* **cli:** add 'workspaces archive' and 'workspaces delete' commands ([5f48347](https://github.com/hop-top/aps/commit/5f4834745016c2d06183889e8f4ae34cc9c0ce4e))
* **cli:** add 'workspaces link' and 'workspaces unlink' commands ([9dc18c6](https://github.com/hop-top/aps/commit/9dc18c642fb6e144ad9e9209f784616829cd8ef4))
* **cli:** add 'workspaces list' command ([af44b2e](https://github.com/hop-top/aps/commit/af44b2e140daa50824892b38c3578b2ad4651e34))
* **cli:** add 'workspaces show' command ([bf94069](https://github.com/hop-top/aps/commit/bf94069fb37ec18e856d7a5973e1a8f12957f504))
* **cli:** add aps status reserved subcommand (T-0681) ([#61](https://github.com/hop-top/aps/issues/61)) ([46a5256](https://github.com/hop-top/aps/commit/46a5256e8e4a85bbbbeac94516e734906d070210))
* **cli:** add aps toolspec subcommand for agent consumption ([9de0dd9](https://github.com/hop-top/aps/commit/9de0dd9566e3602b02e9097ed02b6b3f808e84d2))
* **cli:** add bundle list/show/create/edit/delete/validate subcommands (T-0063..T-0068) ([62044d5](https://github.com/hop-top/aps/commit/62044d5abac77099cde91ce755656d8fb1fba957))
* **cli:** add kit/dry-run-rationale annotations (T-0656) ([#60](https://github.com/hop-top/aps/issues/60)) ([97ad15a](https://github.com/hop-top/aps/commit/97ad15a65655a26951b0ba8aef3cb7c98311cdef))
* **cli:** add kit/examples + kit/next-steps annotations (T-0655) ([#59](https://github.com/hop-top/aps/issues/59)) ([8275419](https://github.com/hop-top/aps/commit/82754196d860e3ac7afd12886945db62a2a5f750))
* **cli:** add workspaces command group with 'new' command ([a59e640](https://github.com/hop-top/aps/commit/a59e6403b129bd4f86fa72dd56a94ede24efa4cc))
* **cli:** adopt kit/console/progress for long-running ops (T-0650) ([#76](https://github.com/hop-top/aps/issues/76)) ([ea07178](https://github.com/hop-top/aps/commit/ea07178e78799b24081b136305e3e626b6001484))
* **cli:** annotate 9 top-level verbs (T-0679) ([#63](https://github.com/hop-top/aps/issues/63)) ([52ba18c](https://github.com/hop-top/aps/commit/52ba18c4040d5409a4ff2badc7f078407aae931d))
* **cli:** auto-enable protocols on first server start ([b4887e3](https://github.com/hop-top/aps/commit/b4887e396d737aff016c3d475f25aafe71527e1b))
* **cli:** declare tool-level globals --config/--profile/--workspace (T-0376) ([d02744f](https://github.com/hop-top/aps/commit/d02744fb428fd89b31ea13cd8874fb8f2d229195))
* **cli:** enable Layer-A strict validation at boot (T-0662) ([#68](https://github.com/hop-top/aps/issues/68)) ([0fea0f0](https://github.com/hop-top/aps/commit/0fea0f0870c914cbc3508007577ac6910015291e))
* **cli:** idemstore + §8.6 delegation policy on mutating cmds (T-0468, T-0469) ([#75](https://github.com/hop-top/aps/issues/75)) ([6690792](https://github.com/hop-top/aps/commit/66907923395772a7af5b6367bd43309a11d0a792))
* **cli:** implement protocol toggle commands for A2A, ACP, and Webhooks ([4fb2776](https://github.com/hop-top/aps/commit/4fb27763a841d37d7f96945e64a55e5307d27e1c))
* **cli:** map domain errors to canonical exit codes ([12f2316](https://github.com/hop-top/aps/commit/12f2316a62bbec6394c924f34b521b94a3a43c8b))
* **cli:** migrate non-list tabwriter callsites to styled tables ([970cbdc](https://github.com/hop-top/aps/commit/970cbdcf0ac9707acbd0e34600722f991a2cf77a))
* **cli:** profile capability commands + rich show ([da59b81](https://github.com/hop-top/aps/commit/da59b81097a5ba0d14d68535df31526f305ae63d))
* **cli:** raise MaxHierarchyDepth to 4 for adapter messenger link (T-0680) ([#62](https://github.com/hop-top/aps/issues/62)) ([b2628a6](https://github.com/hop-top/aps/commit/b2628a695d84fa78d83e32ede62c8052bf2f25ae))
* **cli:** replace per-cmd --json with persistent --format (T-0345) ([89cf123](https://github.com/hop-top/aps/commit/89cf1239ac08269dc57731b1be784cd1ff6832c9))
* **cli:** show workspace info in session list and profile show ([51f59e2](https://github.com/hop-top/aps/commit/51f59e263c5f085ef9e3e3049e902232b6d30e7c))
* **cli:** T-0648 kit 0.4 signature-validator conformance (umbrella) ([#56](https://github.com/hop-top/aps/issues/56)) ([2aed851](https://github.com/hop-top/aps/commit/2aed8513309105f208a1d86694b58c00d9ac49e6))
* **cli:** UX polish — lipgloss tables, huh prompts, error consistency ([#15](https://github.com/hop-top/aps/issues/15)) ([91f3e88](https://github.com/hop-top/aps/commit/91f3e888692e3ca241ddec81b4a781e38f732fa2))
* **cli:** wire aps bundle subcommand into root CLI (T-0069) ([b475387](https://github.com/hop-top/aps/commit/b47538720038af162b45ab60f8378ca432705fdc))
* **cli:** wire destructive-token confirm flow (T-0654) ([#58](https://github.com/hop-top/aps/issues/58)) ([9ecc306](https://github.com/hop-top/aps/commit/9ecc3068d13e6d9209b990ac1274d2e1a22077d1))
* **cli:** wire kit config path/paths subcommands (T-0457) ([5337d0c](https://github.com/hop-top/aps/commit/5337d0ce870bfb77993f6c8049380c84d21fe28c))
* **cli:** wire kit hint system for post-command suggestions (T-0346) ([e685999](https://github.com/hop-top/aps/commit/e685999e0882c707bf131182d83d7e459bcd1435))
* **cli:** wire kit slog handler (T-0647) ([#45](https://github.com/hop-top/aps/issues/45)) ([41a34f4](https://github.com/hop-top/aps/commit/41a34f430bac16c6cd3390aeadf85cd958ff6687))
* **cli:** wire kit/cli HelpConfig groups + assign every command (T-0366, T-0367) ([fad905d](https://github.com/hop-top/aps/commit/fad905dfd8b25b7dba7713c54e6490ab35fa2a91))
* **collab:** implement multi-agent collaboration system ([5297a18](https://github.com/hop-top/aps/commit/5297a18f159758b35f14e9bcb708fdbfb49b743c))
* complete A2A protocol integration with full CLI support ([4605ce9](https://github.com/hop-top/aps/commit/4605ce91db73ad0e028a544a416b042458840e6b))
* **config:** wire kit -c/--config flag (T-0583) ([e594429](https://github.com/hop-top/aps/commit/e59442986f7968a06d9394299d5ca3621abe8ca3))
* consolidate aps chat lanes ([1fcb512](https://github.com/hop-top/aps/commit/1fcb512b0f79fdff8e466d1f5e9f9ac11b6d73b1))
* consolidate aps chat lanes ([7543267](https://github.com/hop-top/aps/commit/7543267ea984f3e5d4cefad0ea8c2cd7a68a4e23))
* **contacts:** contact adapter + cardamum backend + CLI ([48de285](https://github.com/hop-top/aps/commit/48de28526a68e13c222dbb79233e2459c1b2ea42))
* **core:** add instance resolver for --instance global (T-0412) ([f29b8aa](https://github.com/hop-top/aps/commit/f29b8aa40a90f851df6244aec67d48a8a6219d41))
* **core:** add shared styles + capability type system ([b3904dd](https://github.com/hop-top/aps/commit/b3904dd4fba140f4c5763a1fa7d04394d88fa91a))
* **core:** add WorkspaceLink to Profile and WorkspaceID to SessionInfo ([592091d](https://github.com/hop-top/aps/commit/592091d60a4ce9e3d7de4723fd87b8c100d3f29f))
* **core:** adopt hop.top/uri for profile identity ([2fe16e0](https://github.com/hop-top/aps/commit/2fe16e093924b1dfc47e756484d62b7fa062e952))
* **core:** per-profile capability injection ([b4f4c55](https://github.com/hop-top/aps/commit/b4f4c5561bc648c9601c4b20da1391ad889497b3))
* **device:** implement device capability framework ([ad9ec3d](https://github.com/hop-top/aps/commit/ad9ec3d900ed435e133800ea0392b7f1e66b45e2))
* **device:** implement mobile device linking via QR code ([c5f4df8](https://github.com/hop-top/aps/commit/c5f4df84dbe817103f88283a4b046dd31e74231f))
* **domain:** adopt domain.Entity on Profile + ProfileRepo ([e88cb42](https://github.com/hop-top/aps/commit/e88cb427d04159be113d3f1511a5c6a4810ec66e))
* **domain:** adopt domain.StateMachine for session status transitions ([4b274a6](https://github.com/hop-top/aps/commit/4b274a6cbe78590ef4754eeeaea25a1b0a397a94))
* **events:** add bus-backed event types and publisher ([a445ad6](https://github.com/hop-top/aps/commit/a445ad6da761b6af0087905eb7d2da008bc8aaa9))
* **events:** add session/webhook/action topic constants + tests ([35eedbc](https://github.com/hop-top/aps/commit/35eedbcc1b1629cf57f55fe458f2a498a7d73903))
* **events:** emit profile lifecycle events from CLI commands ([1738225](https://github.com/hop-top/aps/commit/1738225d00ebe4fd0f7818d2c95188582957412b))
* **events:** emit profile lifecycle events from core ([083a1ba](https://github.com/hop-top/aps/commit/083a1baf5653e3f2fbda7df54ab5968cf61bb218))
* **events:** emit session lifecycle events from registry ([d58f0f3](https://github.com/hop-top/aps/commit/d58f0f338e2ec538a642622997faf742c8e45e8d))
* execute CLI skill scripts ([d49e41e](https://github.com/hop-top/aps/commit/d49e41e5376e3ee1053ea6bd9a1a95a93be0ddca))
* execute CLI skill scripts ([9f2736f](https://github.com/hop-top/aps/commit/9f2736ff70569a4a3e3b5d43c8293d1714b0b0e2))
* **ibr:** add IBR capability and update module paths ([f521500](https://github.com/hop-top/aps/commit/f52150008734f6e0b50bd860da7fca9b931f5851))
* implement ACP skill invocation ([fbf91ef](https://github.com/hop-top/aps/commit/fbf91ef4ce4cfc20832abb896a93148a589b5529))
* implement ACP skill invocation ([db09057](https://github.com/hop-top/aps/commit/db090572a82205fb16b051d5bd5bdd61ff1f02c6))
* implement Agent Skills support with cross-platform adapters ([1172a7e](https://github.com/hop-top/aps/commit/1172a7eecfcad58bca1969579fc676763e6fd3b5))
* implement capability management, release automation, and docs restructure ([294ac13](https://github.com/hop-top/aps/commit/294ac134a85ad13220eb88f77ca03845898669e9))
* **listen:** aps listen --profile minimal subscribe + dispatch ([280c3ef](https://github.com/hop-top/aps/commit/280c3ef894d1186dab6a2883ea702ef126552c90))
* **listing:** plumb kit TableStyle through RenderList ([e61603e](https://github.com/hop-top/aps/commit/e61603e853c63c001aaa8bb6ea892ea5526722e4))
* **messenger:** add ReplyDestination + SideChatLifecycle hints to ChatTurnResult ([#70](https://github.com/hop-top/aps/issues/70)) ([8daa96b](https://github.com/hop-top/aps/commit/8daa96bebe8e355ad15097c689041437f4b402e2))
* **messenger:** implement messenger-device integration ([deb7a83](https://github.com/hop-top/aps/commit/deb7a8351c81616b4e8283eabc4862da31c2b7ca))
* **multidevice:** implement multi-device workspace access ([e1b812d](https://github.com/hop-top/aps/commit/e1b812d69845372375ea06b195c51a7acceedbde))
* **policy:** cross-agent context delete requires owner (T-1302) ([fec8e45](https://github.com/hop-top/aps/commit/fec8e45c5fa5ff9df7bce2f1071552773a7f9be4))
* **policy:** wire kit/runtime/policy + ship trivial defaults (T-1292) ([7bc5209](https://github.com/hop-top/aps/commit/7bc5209499ae0952468f4edb744fa1abca7db26e))
* **policy:** workspace-aware principal resolver (T-1308) ([c5e6037](https://github.com/hop-top/aps/commit/c5e6037bc73e6943f64b8ca6ddee4492c70578ef))
* **profile:** add optional avatar + color with auto-assign ([7c7e1c2](https://github.com/hop-top/aps/commit/7c7e1c2c7225e611966baa3fce5a5e629f41e896))
* **profile:** add roles and trust ledger ([a850beb](https://github.com/hop-top/aps/commit/a850bebbab30160f08a7c6e8d375f5f2751d2f5f))
* **run:** propagate child process exit code (T-0570) ([98aeaa8](https://github.com/hop-top/aps/commit/98aeaa80dd04ee3f3a20c4070eb71d6a0b501412))
* **run:** wire kit/console/progress on aps run ([713c5e5](https://github.com/hop-top/aps/commit/713c5e5631a34b0301bfead3be7bc25b3b1ee641))
* **scope:** implement unified scope type with intersection logic (T-0014..T-0018) ([211aa87](https://github.com/hop-top/aps/commit/211aa878ef41d905a4b992603fa7fadf37fcfb80))
* **secrets:** adopt kit/storage/secret for profile secrets ([d406fd5](https://github.com/hop-top/aps/commit/d406fd554785557defd39ae11fa672d51e3aa8c2))
* **security:** wire kit/core/redact at logger + output boundaries (T-0460) ([36e2888](https://github.com/hop-top/aps/commit/36e28885150022a4c7783bc87eca52d33a62ab7f))
* **session:** add SessionType field to SessionInfo (T-0364) ([46f8b71](https://github.com/hop-top/aps/commit/46f8b71353bdd27ac83c0068c8f80593780656a7))
* **site:** Astro+Starlight site with Go vanity redirect for hop.top/aps ([e01e239](https://github.com/hop-top/aps/commit/e01e239696716811d6e42de2f12943a0bacc4b45))
* **squad:** add contracts, router, exit conditions, timebox, evolution, checklist, CLI check (T-0031..T-0039) ([9f8f16b](https://github.com/hop-top/aps/commit/9f8f16b45d4e86b0d66b98af7fe8132107fdc76c))
* **squad:** implement squad core — types, manager, scope, CLI, profile membership, export (T-0007..T-0013) ([aa963d5](https://github.com/hop-top/aps/commit/aa963d531b781fc84e8f5415a4eb6aafb0795c50))
* **storage:** sqlstore-backed capability cache ([49944b7](https://github.com/hop-top/aps/commit/49944b721566d1e834339b1a32f51a6e3ef0863a))
* support adapter create device types ([08fe9d2](https://github.com/hop-top/aps/commit/08fe9d25a86881a7b9bea9174b996e99f1dc3581))
* support adapter create device types ([09873fc](https://github.com/hop-top/aps/commit/09873fcbf795660354e8bc15428b85a085a11ccf))
* **tui:** add capability management states ([76303af](https://github.com/hop-top/aps/commit/76303affb341f41e5fcf9269c2bdcc9ae59edc3a))
* **upgrade:** integrate hop.top/upgrade into aps ([f2f3e20](https://github.com/hop-top/aps/commit/f2f3e201201e95b18ce9f3aa13440c0ebe24d70f))
* **upgrade:** merge feat/upgrade into main ([a18d5ca](https://github.com/hop-top/aps/commit/a18d5ca6424921a7534cda7082b97a97d7985637))
* **voice:** add voice config types and Profile.Voice field ([fe70b23](https://github.com/hop-top/aps/commit/fe70b2357914bb90bdce6859fd38db53ab44941f))
* **voice:** add voice config types and Profile.Voice field ([3457b8a](https://github.com/hop-top/aps/commit/3457b8ac70c1e78cb28b2f5298c2f49ec0dbf461))
* **voice:** backend manager with process lifecycle and auto-detection ([0f56ab0](https://github.com/hop-top/aps/commit/0f56ab0c672f7b38531a0a9261a51edd5105c702))
* **voice:** ChannelAdapter and ChannelSession interfaces ([4cb9c77](https://github.com/hop-top/aps/commit/4cb9c77cfb8b58710de727faf9c22c6158c44ac9))
* **voice:** CLI commands for voice service and session management ([60ebfa0](https://github.com/hop-top/aps/commit/60ebfa0dd024d67b5c4731bf7013f3cbc4c36773))
* **voice:** messenger channel adapter for Telegram/WhatsApp voice messages ([d58488b](https://github.com/hop-top/aps/commit/d58488be506e2098ea988573546a66776138d4ed))
* **voice:** persona prompt auto-generation from Profile fields ([edb2f0d](https://github.com/hop-top/aps/commit/edb2f0ddd76a9766c91f61cae1bd838b0282394a))
* **voice:** session manager with create/list/close/switch ([3d06a71](https://github.com/hop-top/aps/commit/3d06a71dc6a55da5eb213f3741ac0caf05bd2001))
* **voice:** TUI Unix socket channel adapter ([2a0c4ba](https://github.com/hop-top/aps/commit/2a0c4baa03c4925ccb55b437fb504957bf6834c2))
* **voice:** Twilio Media Streams channel adapter ([de2f5df](https://github.com/hop-top/aps/commit/de2f5dfdc2a338bf4a4a31d99ca551d521bae2c8))
* **voice:** web channel adapter with WebSocket upgrade ([75d8292](https://github.com/hop-top/aps/commit/75d8292379b1831e7dafd070d3d1e06f395fde90))
* wire ACP MCP bridge tools ([7b28438](https://github.com/hop-top/aps/commit/7b28438cfcfdac3c2c0fc59b40c609de236a9084))
* wire ACP MCP bridge tools ([ce4776b](https://github.com/hop-top/aps/commit/ce4776b86b732a717d61cb1c650a865928298e4e))
* **workspace:** add adapter wrapping wsm Manager ([dc9ba33](https://github.com/hop-top/aps/commit/dc9ba33391c7bcab5e3edc3b8f9982d9a1c8c820))
* **workspace:** add ContextVariable.Visibility (T-1309) ([be2d6a8](https://github.com/hop-top/aps/commit/be2d6a84633eaedd9f31c2104d1f153aabf797e6))
* **workspace:** add profile-workspace linking functions ([9f93214](https://github.com/hop-top/aps/commit/9f9321454a1258a6054d1b30a97e663b806ffe4e))
* **workspace:** add unarchive command ([153caf3](https://github.com/hop-top/aps/commit/153caf3baf7688957f3fdf3a7bd85f8e99176046))
* **wsm:** integrate with wsm ([6a77080](https://github.com/hop-top/aps/commit/6a7708071effe51b43b7b81e2e33251061fd5673))


### Bug Fixes

* **adapter:** persist LinkedTo in adapter manifest ([25a9b57](https://github.com/hop-top/aps/commit/25a9b573deb00e107da9acbb3731095dc375f2fa))
* address Linux-only test flakes exposed by PR [#41](https://github.com/hop-top/aps/issues/41) (T-0677) ([#44](https://github.com/hop-top/aps/issues/44)) ([cd5460b](https://github.com/hop-top/aps/commit/cd5460b620e21e647146495a2fcc17cbc76d87d7))
* address session and profile lifecycle gaps ([#17](https://github.com/hop-top/aps/issues/17)) ([c1bebb2](https://github.com/hop-top/aps/commit/c1bebb27599aa8341169482d604c25f60ddb98bf))
* **bus:** drain async forwarder before short-lived CLI exits ([4c79fac](https://github.com/hop-top/aps/commit/4c79facf871d1a8136bbed887550a0439880087f))
* **bus:** replace hardcoded dev token with env-based auth ([a63aeea](https://github.com/hop-top/aps/commit/a63aeea89e52ce806dc4481f23feea4815ccd89a))
* **ci:** bump Go to 1.26.1 and resolve all CI failures ([#15](https://github.com/hop-top/aps/issues/15)) ([#13](https://github.com/hop-top/aps/issues/13)) ([b522729](https://github.com/hop-top/aps/commit/b522729f9d7de0fcdbd3a2ac11d736ff696384c0))
* **ci:** update license to proprietary and install syft for SBOM ([ae31a35](https://github.com/hop-top/aps/commit/ae31a35853856d85070f8a99eba7ef890653c567))
* **cli:** add skill to PIPELINES group ([e06ed1c](https://github.com/hop-top/aps/commit/e06ed1c0ec37ec9b1e2363da6c9f6395af743d96))
* **cli:** parity re-check round (T-0386, T-0390, T-0394, T-0396, T-0400) ([f701c2c](https://github.com/hop-top/aps/commit/f701c2c81bd6232c3237cabb277a09ac181a31bf))
* **cli:** parity re-check round, kit adoption (T-0388, T-0392, T-0398) ([6384c04](https://github.com/hop-top/aps/commit/6384c0492a25e86e9f3b92bacccb092563d8bc14))
* **cli:** reconcile post-merge build errors ([20459fe](https://github.com/hop-top/aps/commit/20459fe811f194708162b2d692b3cccd5c27c87b))
* **cli:** restore secrets-present detection by file existence ([3a2b597](https://github.com/hop-top/aps/commit/3a2b597f2300913277b6a25f87ba19e4ead323a4))
* **cli:** sweep uncommitted leftovers from phase 2/5 parallel work ([548da2e](https://github.com/hop-top/aps/commit/548da2e10f52bb730ee47439aa9a704c5563a176))
* **core/chat:** security + correctness hardening (cherry-pick from [#39](https://github.com/hop-top/aps/issues/39)) ([#69](https://github.com/hop-top/aps/issues/69)) ([07aa4e1](https://github.com/hop-top/aps/commit/07aa4e1ab32dc93932fdbc43dc3f459817ac9330))
* **deps:** adopt a2a-go v0.3.15 TaskStore + RequestHandler API ([#84](https://github.com/hop-top/aps/issues/84)) ([6f66ef2](https://github.com/hop-top/aps/commit/6f66ef21507030c771246e66104ab6b0007b92c9))
* **redact:** wrap 13 deferred HTTP encoder + persisted writer sites (T-0683) ([#79](https://github.com/hop-top/aps/issues/79)) ([944f762](https://github.com/hop-top/aps/commit/944f762673db79dde7fa3e0adda63c090448c3ff))
* remove tracked vendor symlink (hop-top/git#T-0087) ([3e36e77](https://github.com/hop-top/aps/commit/3e36e77438d1b47b1a1b08736f219a1a17be8423))
* rename callsites + globalize workspace/profile flags ([ae0d332](https://github.com/hop-top/aps/commit/ae0d332bc31213361a79504bfa385a512027cb9d))
* resolve failing tests and enable CGO in Makefile ([a457af1](https://github.com/hop-top/aps/commit/a457af1bc3ce7b384556e15307e61740f05f0511))
* **run:** pass through TTY stdout/stderr unwrapped ([ca337dd](https://github.com/hop-top/aps/commit/ca337ddbdf38a7ebcd7a47de601f5a00241999e5))
* **security:** wire --no-redact through kitcli Hooks; expand allowlist (T-0460) ([6fae461](https://github.com/hop-top/aps/commit/6fae461b9371891e9814cb6427375e54bb3f0170))
* set ACP websocket read header timeout ([17c471f](https://github.com/hop-top/aps/commit/17c471fb7e08db7f8ef060f411419ef65f0e70f3))
* **test:** align A2A tests with capability-based enablement guards ([28fee7c](https://github.com/hop-top/aps/commit/28fee7cf7d4e871f0f7c3277d0582714b701d795))
* **test:** align capability E2E assertions with styled CLI output ([1e81483](https://github.com/hop-top/aps/commit/1e81483196005caa28940ade834ce96aa1a8d079))
* **test:** align skills test profile paths with XDG data dir ([1a36ad1](https://github.com/hop-top/aps/commit/1a36ad1075032f1b2ed1a07c40061b0e9a280742))
* **test:** randomize agent_protocol test ports (T-0584) ([b6ab071](https://github.com/hop-top/aps/commit/b6ab071436ed26dbc447ecb5cb81f8b4736abbf6))
* **test:** setup test profile in LinuxSandbox unit tests ([a3b87f0](https://github.com/hop-top/aps/commit/a3b87f0d36feb17b154ae99d028b40aedced0b95))
* **tests:** resolve all failing unit and e2e tests ([91e60dd](https://github.com/hop-top/aps/commit/91e60ddd406797f8e72d8d6ac9d0e448d885d749))
* update module path from oss-aps-cli to hop.top/aps and fix Device→Adapter renames ([8487f50](https://github.com/hop-top/aps/commit/8487f50d57292ade55d4d99d83c4cd6e702de912))


### Refactoring

* **cli:** drop top-level aps messenger, keep under adapter (T-0363) ([0eb43ca](https://github.com/hop-top/aps/commit/0eb43ca38c1bff8d134d5fad9a5aa45f081b87ac))


### Miscellaneous

* release 0.6.0-alpha.0 ([e109fd2](https://github.com/hop-top/aps/commit/e109fd21539bd03a87a0d7c2a192b102fdeae682))

## Changelog

All notable changes to `aps` are documented in this file.

## Unreleased

### Added — per-invocation env overrides for `aps run` (track: `aps-run-env-flag`, T-0576..T-0580)

- `--env KEY=VAL` (repeatable) and `--env-file PATH` (repeatable) flags
  on `aps run` for per-invocation environment overrides.
- Precedence (lowest → highest, last wins on duplicate keys): parent
  `os.Environ()` → profile-injected vars → `--env-file` entries (flag
  order) → `--env` entries (flag order).
- Validation: `--env` requires `KEY=VAL` with a shell-safe key;
  `--env-file` is fatal on missing path; malformed dotenv lines fail
  with the file + line number.
- Dotenv parser accepts blank lines, `#` comments, optional `export`
  prefix, and matching single/double quotes around values. No shell
  escape expansion.
- Redaction guarantee: override values flow through the existing child
  stdout/stderr redacting boundary (story 058), so an `OPENAI_API_KEY`
  passed via `--env` never reaches aps-side logs even though the child
  process legitimately receives the unredacted value. See
  [docs/stories/064-run-env-overrides.md](docs/stories/064-run-env-overrides.md).

### Added — aps-chat messenger bridge (track: `aps-chat`, T-0586/T-0424/T-0425)

**Added**

- Message services can opt into native chat execution with
  `options.execution: chat`. In that mode the shared provider runtime routes
  normalized message handoffs through a `ChatTurnRunner` bridge instead of
  action stdout, and delivers assistant replies through the existing provider
  `DeliveryRequest` path.
- Telegram has focused fake-transport coverage proving inbound webhook text is
  normalized, keyed with `NormalizedMessage.ConversationState().SessionID`,
  handed to the chat bridge, and returned via Bot API `sendMessage`.

**Documented pending integration**

- The planned `aps chat <profile-id>` CLI surface includes the single-profile
  REPL, `--once` one-shot mode, `--invite` multi-profile mode, profile
  `llm:` config, and `SessionTypeChat` registry entries. Those pieces depend
  on the companion aps-chat lanes that own `internal/cli/chat`,
  `internal/core/chat`, and `internal/core/session`.

### Added — kit/runtime/policy adoption (track: `aps-policy-adoption`, T-1290..T-1293)

`aps` now adopts kit's runtime policy engine
(`hop.top/kit/go/runtime/policy`) to gate destructive state changes.

**Added**

- `--note|-n` flag on every state-changing subcommand (52 entries
  spanning profile, identity, session, workspace, capability,
  bundle, squad, adapter, and directory groups). Inventory in
  [docs/cli/reference.md](docs/cli/reference.md); CI verification
  via `scripts/verify_note_flag.sh`.
- Policy adoption from `kit/runtime/policy`: bundled defaults at
  `internal/config/policies_default.yaml`, seeded into
  `$XDG_CONFIG_HOME/aps/policies.yaml` on first boot. Override
  path via `$APS_POLICY_FILE`.
- Bus event payloads carry the `--note` value (e.g.
  `SessionStoppedPayload.Note`, `ProfileDeletedPayload.Note`,
  `AdapterUnlinkedPayload.Note` — see `internal/events/events.go`).
- Cross-agent shared workspace-context delete now requires
  `principal.role == owner` (default policy
  `cross-agent-context-delete-requires-owner`, T-1302). Private
  variables remain exempt — the storage-layer visibility filter
  (T-1309) already keeps them invisible to non-writers, so reaching
  the delete path means the variable belongs to the caller. Decision
  table in [docs/policies.md](docs/policies.md#t-1302-decision-table).
- Documentation: [docs/policies.md](docs/policies.md) (policy
  engine, default rules, anatomy, troubleshooting) and
  [docs/cli/reference.md](docs/cli/reference.md) (global flags,
  exit codes, full `--note` inventory).

**Changed**

- `SessionManager` and `WorkspaceContext` are now backed by
  `hop.top/kit/go/runtime/domain.Service` (T-1290 refactor).
  Operator-visible behavior is the same except that mutations now
  publish `kit.runtime.entity.pre_validated` and
  `kit.runtime.entity.pre_persisted` events on the bus before any
  state change. Successful mutations still fan out the existing
  `aps.session.*` and `aps.profile.*` notification topics.

**Operator note (transition)**

Existing scripts that called the destructive subcommands without
`--note` exit 4 after upgrade. The default policy file ships two
rules:

- `delete-session-requires-note` — blocks `aps session delete`
- `delete-workspace-context-requires-note` — blocks `aps workspace ctx delete`

Upgrade path:

1. **Recommended** — pass `--note "<reason>"` on every state
   change. The note flows into bus event payloads for audit and is
   exposed to policy CEL as `context.note`.

   ```bash
   aps session delete sess-7c41 --note "stale; profile retired"
   aps workspace ctx delete CHANNEL_ID -n "redacted PII"
   ```

2. **Per-shell bypass with the engine still loaded** — point
   `$APS_POLICY_FILE` at an empty policies file. Useful in CI
   and tests because the engine still exercises the load + parse
   path; only rule evaluation is empty:

   ```bash
   echo "policies: []" > /tmp/aps-policies-empty.yaml
   export APS_POLICY_FILE=/tmp/aps-policies-empty.yaml
   ```

3. **Emergency disable** — set `KIT_POLICY_DISABLE=1` to
   short-circuit the engine bootstrap entirely (no load, no
   rules evaluated). Intended for emergency operator override
   and CI debugging; not recommended for normal use:

   ```bash
   KIT_POLICY_DISABLE=1 aps session delete <id>
   ```

4. **Permanent loosening** — edit
   `$XDG_CONFIG_HOME/aps/policies.yaml` and remove or adjust the
   default rules. Aps does not re-seed once a user file exists, so
   the change persists across upgrades.

The `cross-agent-context-delete-requires-owner` rule (T-1302) is
now in the bundled defaults — see "Added" above. Read/list of
shared variables remain ungated; visibility filtering for private
variables happens at the storage layer.

### Improvements — aps-tabwriter-sweep (track: `aps-tabwriter-sweep`)

Five additional `tabwriter.NewWriter` callsites — surfaced as scope
creep during the Wave 2 audit migration (story 059) and held back —
now route through `listing.RenderList`. After this ships,
`grep -rn "tabwriter.NewWriter" internal/ cmd/` returns zero;
every tabular list in aps inherits the kit-themed styled renderer
on TTY writers.

**Migrated callsites** (T-0473..T-0477):

  aps workspace activity   event log table
  aps device presence      device presence table
  aps device pending       pending-approvals table
  aps device channels      channel-discovery table
  aps squad check          topology-validation report

**JSON shape preserved** — `aps device presence --json` keeps
`sync_lag` and `offline_queue` as ints (table path projects to
suffixed strings via a separate `presenceTableRow`). `aps device
pending --json` keeps the exact 5-key field set the prior inline
`pendingDevice` struct emitted.

**Cleanup** — `internal/cli/adapter/styles.go` drops the
`tableHeader = lipgloss.NewStyle()…` var (last consumer migrated)
and the `charm.land/lipgloss/v2` import; header styling now flows
from the active kit/cli theme via the styled renderer. Story 060
documents the sweep and closes the 2026-05-04 kit-integration
audit's §1.1 scorecard ⚠️ row for "Tabular/JSON/YAML render".

### Improvements — kit-styled-table-rollout (track: `kit-styled-table-rollout`)

Tables rendered through `listing.RenderList` and the migrated non-list
callsites now switch to a lipgloss-backed styled renderer when stdout
is a TTY. Non-TTY writers (pipes, files, CI logs) keep emitting the
existing plain tabwriter output; structured (`--format json|yaml`)
output is unchanged.

**Wiring** — `internal/cli/listing.SetTableStyle` installs a default
`output.TableStyle` that `RenderList` forwards via `output.WithTableStyle`.
`internal/cli/root.go` calls it with `kitcli.Root.TableStyle()` during
init so the active CLI theme drives the styled renderer.

**Migrated callsites** (T-0456) — five hand-rolled `tabwriter.NewWriter`
sites cited in the 2026-05-04 kit-integration audit:

  aps profile trust          score + history tables
  aps workspace members      member list
  aps workspace tasks        task list
  aps workspace agents       capability matches
  aps workspace audit        audit trail
  aps workspace ctx history  per-key mutation history
  aps policy list            policy settings table
  aps session inspect        property + environment tables
  aps migrate messengers     dry-run preview table

The shared `workspace/helpers.go:newTabWriter` factory plus the
`tableHeader` lipgloss vars in `policy/cmd.go`, `session/list.go`,
`migrate/cmd.go`, and `workspace/helpers.go` were removed; header
styling now flows from the active theme.

**Coverage** — package-level golden-style tests for each migrated
row type plus `tests/e2e/profile/profile_list_styled_test.go`, which
attaches `aps profile list` to a real pseudo-terminal pair (creack/pty)
and asserts ANSI + box-drawing chars on the TTY path versus plain
output on the non-TTY path with content identity after stripping ANSI.

### Improvements — list-commands-uplift (track: `list-commands-uplift`)

All `aps <noun> list` (and `aps <noun> <subnoun> list`) commands now
emit rich tabular output via kit/output.Render with consistent filter
flag conventions. 16 commands migrated; 1 new shared helper package.

**New** — `internal/cli/listing/`: `RenderList[T]` generic dispatch
to kit/output.Render, `Predicate[T]` combinators (All, Any, Not,
Filter), CLI-flag-shaped helpers (MatchString, MatchSlice, BoolFlag).
24 tests; codifies the pattern in package doc.go.

**Migrated commands** (each now: rich row struct with `priority=N`
tags, json/yaml struct tags, filter flags below, render via
listing.RenderList):

  aps profile list           --capability, --role, --squad,
                             --workspace, --has-identity,
                             --has-secrets, --tone
  aps session list           --type (existing) + --status,
                             --profile, --workspace, --tier
  aps workspace list         --member, --owner, --archived
  aps squad list             --member, --role
  aps bundle list            --tag, --builtin, --user
  aps capability list        --tag, --builtin, --external,
                             --enabled-on
  aps capability patterns    same treatment as parent
  aps contact list           --org, --has-email
                             (existing --addressbook preserved)
  aps adapter list           --type, --status, --workspace
  aps adapter messenger list --platform, --status
  aps adapter links          --profile, --messenger
  aps a2a tasks list         --profile (global), --status
  aps action list            --type (sh|py|js)
  aps skill list             --profile (global), --source
  aps workspace conflicts list  --workspace (global), --unresolved
  aps workspace ctx list     --workspace (global), --key-prefix
  aps workspace policy list  (workspace stays positional;
                             rich row only)

**Removed flags** (canonicalization via shared listing helper):

- `aps skill list --verbose` — kit/output's table priority tags
  drop low-value columns on narrow terminals automatically; full
  fields available via `--format json|yaml`.
- `aps skill list --profile` (local) — superseded by the kit-owned
  global `--profile` (T-0376) bound on root.

**Backing data additions**:

- `core.ListProfilesFull() ([]Profile, error)` — loads each profile
  YAML once; cheap `ListProfiles() []string` retained for callers
  that only need IDs.
- `core/bundle.Bundle.Tags []string` (yaml-tagged; existing assets
  unaffected).
- `core/capability.Capability.Tags`, `BuiltinCapability.Tags`.
- `internal/cli/globals.Profile()` and `globals.Format()` accessors —
  let non-`internal/cli` subpackages read kit-owned globals without
  forming an import cycle.
- `aps skill` now wired to rootCmd (was previously defined but
  unreachable). Lives in PIPELINES group alongside a2a/acp/etc.

**Internal**: `core/skills.Registry.SourceLabel(...)` exported for
the cli row builder (was internal `getSourceLabel`).

Closes the §6 partial-compliance finding in
`~/.ops/reviews/aps-cli-review-2026-04-30.md` (kit/output not fully
adopted across the surface). Convention doc
`~/.ops/docs/cli-conventions-with-kit.md` §3.3 now cites listing.
RenderList as the canonical aps pattern.

### Breaking changes — cli-surface-refactor (track: `cli-surface-refactor`)

This wave consolidates the aps command surface from 30 top-level commands
to 26, drops asymmetric verb-noun-flat commands in favor of noun-verb
subtrees, and adds kit/cli help-section grouping. Aps is pre-release;
no deprecation aliases ship.

**Command merges** (T-0362, T-0363, T-0364):

  aps collab     → aps workspace             (full surface)
  aps audit log  → aps workspace audit
  aps conflict   → aps workspace conflicts
  aps messenger  → aps adapter messenger     (subcommand of adapter)
  aps voice session list  → aps session list --type voice
                            (voice sessions unified into core
                            session registry; SessionType field
                            added to SessionInfo)

**Verb-noun-flat → noun-verb renames** (T-0365, T-0373):

  aps profile add-capability     → aps profile capability add
  aps profile remove-capability  → aps profile capability remove
  aps profile set-workspace      → aps profile workspace set
  aps squad add-member           → aps squad members add
  aps squad remove-member        → aps squad members remove
  aps adapter set-permissions    → aps adapter permissions set
  aps a2a list-tasks             → aps a2a tasks list
  aps a2a get-task               → aps a2a tasks show
  aps a2a send-task              → aps a2a tasks send
  aps a2a cancel-task            → aps a2a tasks cancel
  aps a2a subscribe-task         → aps a2a tasks subscribe
  aps a2a send-stream            → aps a2a tasks stream
  aps a2a fetch-card             → aps a2a card fetch
  aps a2a show-card              → aps a2a card show

**Naming-convention renames** (T-0374, T-0375):

  aps profile new                → aps profile create
  aps directory deregister       → aps directory delete

**Help-section grouping** (T-0366, T-0367) — `aps --help` now organizes
commands into 5 visible groups (INTERACT, ORGANIZE, PIPELINES, SECURITY,
INSTANCE) plus a hidden MANAGEMENT group (alias, docs, env, migrate,
upgrade, toolspec, version, completion). Per-group help via
`--help-<id>`; reveal hidden groups with `--help-all` or
`--help-management`.

**New persistent globals** (T-0376) — `--config`, `--profile`, `--workspace`
declared on the root command. Subcommand-local duplicates removed; reads
fall through cobra's persistent flag set.

### Breaking changes — kit-reorg-adoption (track: `kit-reorg-adoption`)

- **Removed per-command `--json` flags** on `aps version`, `aps profile list`,
  and `aps action list`. Use the persistent `--format` flag (now provided by
  `hop.top/kit/go/console/cli`) to select output mode:
  - `--format table` (default) — human-readable table output
  - `--format json`              — JSON
  - `--format yaml`              — YAML

  Migration: replace `aps version --json` with `aps version --format json`,
  `aps profile list --json` with `aps profile list --format json`, etc.
  Tracked in T-0345 (track: `kit-reorg-adoption`).

- **Flag shortname realignment** (T-0347, audit at
  `docs/plans/2026-04-29-kit-reorg-adoption/flag-audit.md`):
  - Dropped `-f` shortname on `--force` for `profile delete`,
    `session delete`, `session terminate`, and collab helpers (`-f` is
    reserved for a future `--format` short alias). Long form `--force`
    still works.
  - Dropped `-v` shortname on `--verbose` for `profile status`,
    `skill list`, `adapter status`, `adapter links`. Use kit's persistent
    `-V` (count) flag instead, or the long `--verbose` form locally.
  - Removed `aps upgrade -q --quiet`. The flag is now provided by kit as a
    persistent root flag — `aps upgrade --quiet` still works, but reads
    from `root.Viper.GetBool("quiet")` instead of a local flag.

### Added

- `-n` short alias for `--dry-run` on: `action run`, `adapter link`,
  `adapter unlink`, `adapter stop`, `adapter revoke`, `conflict resolve`,
  `migrate messengers`, `webhook server` (POSIX `make -n` convention).

### Added

- Persistent `--format` and `--no-hints` flags on the root command, wired by
  `hop.top/kit/go/console/cli` via `output.RegisterFlags` and
  `output.RegisterHintFlags` (T-0344).
