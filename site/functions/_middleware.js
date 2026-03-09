/**
 * Cloudflare Pages middleware — Go vanity import redirect for hop.top/aps
 *
 * go-import format: <import-prefix> <vcs> <repo-root>
 */

const MODULE = "hop.top/aps";
const VCS = "git";
const REPO = "https://github.com/hop-top/aps";

export async function onRequest(context) {
  const url = new URL(context.request.url);

  if (url.searchParams.get("go-get") === "1") {
    const html = `<!DOCTYPE html>
<html>
<head>
<meta name="go-import" content="${MODULE} ${VCS} ${REPO}">
<meta name="go-source" content="${MODULE} ${REPO} ${REPO}/tree/main{/dir} ${REPO}/blob/main{/dir}/{file}#L{line}">
</head>
<body>
<p>go get ${MODULE}</p>
</body>
</html>`;

    return new Response(html, {
      status: 200,
      headers: { "Content-Type": "text/html; charset=utf-8" },
    });
  }

  return context.next();
}
