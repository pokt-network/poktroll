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

  url: "https://poktroll.com/",
  baseUrl: "/",

  markdown: { mermaid: true },
  themes: ["@docusaurus/theme-mermaid"],

  organizationName: "pokt-network", // Usually your GitHub org/user name (ORGANIZATION_NAME)
  projectName: "poktroll", // Usually your repo name. (PROJECT_NAME)
  deploymentBranch: "gh-pages", // Deployment branch (DEPLOYMENT_BRANCH)

  trailingSlash: false,
  onBrokenLinks: "throw",
  onBrokenMarkdownLinks: "warn",

  i18n: {
    defaultLocale: "en",
    locales: ["en"],
  },

  presets: [
    [
      "classic",
      // "@docusaurus/preset-classic",
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          sidebarPath: require.resolve("./sidebars.js"),
        },
        theme: {},
      }),
    ],
  ],

  plugins: [
    [
      "@docusaurus/plugin-content-docs",
      {
        id: "docs-deployment",
        path: "docs-deployment",
        routeBasePath: "docs-deployment",
        sidebarPath: require.resolve("./sidebars-deployment.js"),
      },
    ],
    [
      "@docusaurus/plugin-content-docs",
      {
        id: "docs-development",
        path: "docs-development",
        routeBasePath: "docs-development",
        sidebarPath: require.resolve("./sidebars-development.js"),
      },
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      style: "dark",
      navbar: {
        title: "Pocket Network",
        logo: {
          alt: "Pocket Network Logo",
          src: "img/logo.png",
        },
        items: [
          {
            to: "docs-deployment/README",
            label: "⚙️ Deployment",
            position: "left",
            docsPluginId: "docs-deployment",
            activeBasePath: "docs-deployment",
          },
          {
            to: "docs/development",
            label: "👨‍💻 Development",
            position: "left",
            docsPluginId: "developmentPluginId",
            activeBaseRegex: "docs/(next|v8)",
          },
          {
            docsPluginId: "protocolPluginId",
            to: "docs/protocol",
            label: "🧠 Protocol",
            position: "left",
            activeBaseRegex: "docs/(next|v8)",
          },
          {
            docsPluginId: "planningPluginId",
            to: "docs/planning",
            label: "🗒️ Planning",
            position: "left",
            activeBaseRegex: "docs/(.*)",
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
      },
    }),
};

export default config;
