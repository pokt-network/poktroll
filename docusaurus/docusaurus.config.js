// @ts-check
// `@type` JSDoc annotations allow editor autocompletion and type checking
// (when paired with `@ts-check`).
// There are various equivalent ways to declare your Docusaurus config.
// See: https://docusaurus.io/docs/api/docusaurus-config

import { themes as prismThemes } from "prism-react-renderer";

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: "Poktroll",
  tagline: "Roll the POKT",
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

  // GitHub pages deployment config.
  url: "https://poktroll.com/",
  baseUrl: "/",

  // Custom domain config.
  // url: "https://docs.poktroll.com",
  // baseUrl: "/",

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: "pokt-network", // Usually your GitHub org/user name (ORGANIZATION_NAME)
  projectName: "poktroll", // Usually your repo name. (PROJECT_NAME)
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
      // "classic",
      "@docusaurus/preset-classic",
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          // path: "docs",
          routeBasePath: "/",
          sidebarPath: "./sidebars.js",
          sidebarCollapsible: false,
          remarkPlugins: [require("remark-math")],
          rehypePlugins: [require("rehype-katex")],
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
          hideable: false,
          autoCollapseCategories: false,
        },
      },
      // image: "img/docusaurus-social-card.jpg",
      style: "dark",
      navbar: {
        title: "Pocket Network",
        logo: {
          alt: "Pocket Network Logo",
          src: "img/logo.png",
        },
        items: [
          {
            type: "docSidebar",
            position: "left",
            sidebarId: "operateSidebar",
            label: "⚙️ Operate",
          },
          {
            type: "docSidebar",
            position: "left",
            sidebarId: "developSidebar",
            label: "💻 Develop",
          },
          {
            type: "docSidebar",
            position: "left",
            sidebarId: "protocolSidebar",
            label: "🧠 Protocol",
          },
          {
            type: "docSidebar",
            position: "left",
            sidebarId: "exploreSidebar",
            label: "🗺 Explore",
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
                label: "Poktroll",
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
                href: "https://github.com/pokt-network/poktroll",
              },
            ],
          },
        ],
        copyright: `MIT License © Pocket Network`,
      },
      prism: {
        theme: prismThemes.github,
        darkTheme: prismThemes.dracula,
        additionalLanguages: ["gherkin", "protobuf", "json", "makefile"],
      },
    }),
};

export default config;
