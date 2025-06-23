// @ts-check
// `@type` JSDoc annotations allow editor autocompletion and type checking
// (when paired with `@ts-check`).
// There are various equivalent ways to declare your Docusaurus config.
// See: https://docusaurus.io/docs/api/docusaurus-config.

import { themes as prismThemes } from "prism-react-renderer";
import rehypeKatex from "rehype-katex";
import remarkMath from "remark-math";

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: "Pocket",
  tagline: "Permissionless APIs",
  favicon: "img/logo.png",

  markdown: {
    mermaid: true,
  },

  themes: [
    "@docusaurus/theme-mermaid",
    [
      require.resolve("@easyops-cn/docusaurus-search-local"),
      /** @type {import('@easyops-cn/docusaurus-search-local').PluginOptions} **/
      {
        docsRouteBasePath: "/",
        hashed: false,
        indexBlog: false,
        highlightSearchTermsOnTargetPage: true,
        explicitSearchResultPath: true,
      },
    ],
  ],

  plugins: [
    [
      require.resolve("docusaurus-plugin-chat-page"),
      {
        baseURL:
          process.env.NODE_ENV === "development"
            ? "http://localhost:4000"
            : "https://dev.poktroll.com",
        path: "chat",
        openai: {
          apiKey: process.env.OPENAI_API_KEY,
        },
        prompt: {
          systemPrompt: require("./prompt"),
          model: "gpt-4o-mini",
          temperature: 0.7,
          maxTokens: 1000,
        },
        embeddingCache: {
          mode: "auto",
          // mode: "skip",
          // strategy: "manual", // Avoid regeneration every time for speed & price (just a v1)
          path: "embeddings.json",
        },
        embedding: {
          model: "text-embedding-3-small",
          chunkSize: 2000,
          chunkingStrategy: "headers", // Splits at markdown headers!
          batchSize: 5,
          maxChunksPerFile: 15,
          relevantChunks: 5,
        },
      },
    ],
  ],

  // GitHub pages deployment config.
  url: "https://pocket.com/",
  baseUrl: "/",

  // Custom domain config.
  // url: "https://docs.poktroll.com",
  // baseUrl: "/",

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: "pokt-network", // Usually your GitHub org/user name (ORGANIZATION_NAME)
  projectName: "pocket", // Usually your repo name. (PROJECT_NAME)
  deploymentBranch: "gh-pages", // Deployment branch (DEPLOYMENT_BRANCH)
  trailingSlash: false,

  onBrokenLinks: "throw",
  onBrokenMarkdownLinks: "warn",

  // Even if you don't use internationalization, you can use this field to set
  // useful metadata like html lang. For example, if your site is Chinese, you
  // may want to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: "en",
    locales: ["en"],
  },

  presets: [
    [
      "classic",
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          routeBasePath: "/",
          sidebarPath: "./sidebars.js",
          sidebarCollapsible: false,
          remarkPlugins: [remarkMath],
          rehypePlugins: [rehypeKatex],
          editUrl:
            "https://github.com/pokt-network/poktroll/edit/main/docusaurus",
        },
        theme: {
          customCss: [
            require.resolve("./src/css/custom.css"),
            require.resolve("./src/css/header-icons.css"),
          ],
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      docs: {
        sidebar: {
          hideable: true,
          autoCollapseCategories: true,
        },
      },
      style: "dark",
      navbar: {
        // title: "Pocket Network",
        logo: {
          alt: "Pocket Network Logo",
          src: "img/logo.png",
        },
        items: [
          {
            type: "docSidebar",
            position: "left",
            sidebarId: "operateSidebar",
            label: "⚙️ Infra Operators",
            to: "/1_operate/",
          },
          {
            type: "docSidebar",
            position: "left",
            sidebarId: "toolsSidebar",
            label: "🗺 Users & Explorers",
            to: "/2_explore/user_guide/create-new-wallet",
          },
          {
            type: "docSidebar",
            position: "left",
            sidebarId: "protocolSidebar",
            label: "🧠 Protocol Specifications",
            to: "/3_protocol/",
          },
          {
            type: "docSidebar",
            position: "left",
            sidebarId: "developSidebar",
            label: "🧑‍💻️ Core Developers",
            to: "/4_develop/",
          },
          {
            to: "/chat",
            label: "🤖 Chat",
            position: "left",
          },
        ],
      },
      footer: {
        style: "dark",
        links: [
          {
            title: "Documentation",
            items: [
              {
                label: "Pocket",
                to: "/",
              },
              {
                label: "Pocket Network",
                href: "https://docs.pokt.network/",
              },
            ],
          },
          {
            title: "Community",
            items: [
              {
                label: "Discord - Pocket",
                href: "https://discord.gg/6cKbYA2X",
              },
              {
                label: "Twitter",
                href: "https://twitter.com/poktnetwork",
              },
            ],
          },
          {
            title: "More",
            items: [
              {
                label: "GitHub",
                href: "https://github.com/pokt-network/pocket",
              },
            ],
          },
        ],
        copyright: `MIT License © Pocket Network`,
      },
      prism: {
        theme: prismThemes.github,
        darkTheme: prismThemes.dracula,
        additionalLanguages: [
          "gherkin",
          "protobuf",
          "json",
          "makefile",
          "diff",
          "bash",
        ],
      },
    }),
};

export default config;
