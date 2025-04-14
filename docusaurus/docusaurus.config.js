// @ts-check
// `@type` JSDoc annotations allow editor autocompletion and type checking
// (when paired with `@ts-check`).
// There are various equivalent ways to declare your Docusaurus config.
// See: https://docusaurus.io/docs/api/docusaurus-config

import { themes as prismThemes } from "prism-react-renderer";

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: "Pocket",
  tagline: "Permissionless APIs",
  favicon: "img/logo.png",

  markdown: {
    mermaid: true,
  },

  // Custom fields for configuration
  customFields: {
    // Configuration that might be used by plugins or themes
    hotReload: true,
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
      // Add cache control and auto-refresh
      metadata: [
        { name: "cache-control", content: "no-cache" },
        { name: "refresh", content: "3" },
      ],
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
            label: "⚙️ Guides & Deployment",
          },
          {
            type: "docSidebar",
            position: "left",
            sidebarId: "toolsSidebar",
            label: "🗺 Tools & Explorers",
          },
          {
            type: "docSidebar",
            position: "left",
            sidebarId: "developSidebar",
            label: "🧑‍💻️ Core Developers",
          },
          {
            type: "docSidebar",
            position: "left",
            sidebarId: "protocolSidebar",
            label: "🧠 Protocol Specification",
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
