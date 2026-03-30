import { Html, Head, Main, NextScript } from "next/document";

export default function Document() {
  return (
    <Html lang="en">
      <Head>
        <link rel="icon" href="/favicon.ico" sizes="any" />
        <link rel="icon" href="/favicon-32x32.png" type="image/png" sizes="32x32" />
        <link rel="icon" href="/favicon-16x16.png" type="image/png" sizes="16x16" />
        <link rel="apple-touch-icon" href="/apple-touch-icon.png" />
        <meta name="application-name" content="Drawgo" />
      </Head>
      <body className="antialiased">
        <Main />
        <NextScript />
        {process.env.NODE_ENV === "development" && (
          <script dangerouslySetInnerHTML={{ __html: `
            window.addEventListener('error', function(e) {
              try {
                fetch('/api/v1/log-error', {
                  method: 'POST',
                  headers: { 'Content-Type': 'application/json' },
                  body: JSON.stringify({
                    type: 'global-error',
                    message: e.message,
                    filename: e.filename,
                    lineno: e.lineno,
                    colno: e.colno,
                    stack: e.error && e.error.stack
                  })
                }).catch(function(){});
              } catch(ex) {}
            });
            window.addEventListener('unhandledrejection', function(e) {
              try {
                fetch('/api/v1/log-error', {
                  method: 'POST',
                  headers: { 'Content-Type': 'application/json' },
                  body: JSON.stringify({
                    type: 'unhandled-rejection',
                    message: String(e.reason),
                    stack: e.reason && e.reason.stack
                  })
                }).catch(function(){});
              } catch(ex) {}
            });
          `}} />
        )}
      </body>
    </Html>
  );
}
