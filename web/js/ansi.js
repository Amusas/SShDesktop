// Parseador y desinfectador de secuencias de escape ANSI / VT100
const AnsiParser = (() => {
    const FG_COLORS = [
        'ansi-fg-black', 'ansi-fg-red', 'ansi-fg-green', 'ansi-fg-yellow',
        'ansi-fg-blue', 'ansi-fg-magenta', 'ansi-fg-cyan', 'ansi-fg-white',
    ];
    const BG_COLORS = [
        'ansi-bg-black', 'ansi-bg-red', 'ansi-bg-green', 'ansi-bg-yellow',
        'ansi-bg-blue', 'ansi-bg-magenta', 'ansi-bg-cyan', 'ansi-bg-white',
    ];
    const BRIGHT_FG = [
        'ansi-bright-fg-black', 'ansi-bright-fg-red', 'ansi-bright-fg-green',
        'ansi-bright-fg-yellow', 'ansi-bright-fg-blue', 'ansi-bright-fg-magenta',
        'ansi-bright-fg-cyan', 'ansi-bright-fg-white',
    ];

    // Comandos de sistema operativo (titulos de ventana: \x1b]0;titulo\x07)
    const OSC_REGEX = /\x1b\][^\x07\x1b]*(?:\x07|\x1b\\|\r|\n)/g;

    // Secuencias de modo privado DEC (control de cursor: \x1b[?25h, \x1b[?25l)
    const DEC_REGEX = /\x1b\[\?[0-9;]*[a-zA-Z]/g;

    // Secuencias de control CSI que no sean estilo/color (movimiento de cursor, etc.)
    const NON_SGR_CSI_REGEX = /\x1b\[[0-9;]*[A-LN-Za-ln-z]/g;

    // Patron global para remover todas las secuencias ANSI
    const ALL_ANSI_REGEX = /\x1b(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~]|\][^\x07\x1b]*(?:\x07|\x1b\\|\r|\n))/g;

    function parseSGR(params, classes) {
        if (!params || params === '0') {
            return [];
        }

        const codes = params.split(';').map(Number);
        const result = [...classes];
        let i = 0;

        while (i < codes.length) {
            const code = codes[i];

            if (code === 0) {
                result.length = 0;
            } else if (code === 1) {
                result.push('ansi-bold');
            } else if (code === 2) {
                result.push('ansi-dim');
            } else if (code === 3) {
                result.push('ansi-italic');
            } else if (code === 4) {
                result.push('ansi-underline');
            } else if (code >= 30 && code <= 37) {
                result.push(FG_COLORS[code - 30]);
            } else if (code >= 40 && code <= 47) {
                result.push(BG_COLORS[code - 40]);
            } else if (code >= 90 && code <= 97) {
                result.push(BRIGHT_FG[code - 90]);
            } else if (code === 39) {
                for (let j = result.length - 1; j >= 0; j--) {
                    if (result[j].startsWith('ansi-fg-') || result[j].startsWith('ansi-bright-fg-')) {
                        result.splice(j, 1);
                    }
                }
            } else if (code === 49) {
                for (let j = result.length - 1; j >= 0; j--) {
                    if (result[j].startsWith('ansi-bg-')) {
                        result.splice(j, 1);
                    }
                }
            }
            i++;
        }

        return result;
    }

    function stripAnsi(text) {
        if (!text) return '';
        return text.replace(ALL_ANSI_REGEX, '');
    }

    // Detectar secuencias ANSI de borrado de pantalla (cls / clear)
    function hasClearScreen(text) {
        if (!text) return false;
        return text.includes('\x1b[2J') || text.includes('\x1b[3J');
    }

    function removeClearScreen(text) {
        if (!text) return '';
        return text.replace(/\x1b\[[23]J/g, '').replace(/\x1b\[(?:H|1;1H|2;0H)/g, '');
    }

    function sanitizeText(text) {
        if (!text) return '';
        return text
            .replace(OSC_REGEX, '')
            .replace(DEC_REGEX, '')
            .replace(NON_SGR_CSI_REGEX, '');
    }

    function renderToHTML(text, activeClasses) {
        const cleanText = sanitizeText(text);

        let html = '';
        let classes = [...activeClasses];
        let lastIndex = 0;

        const regex = /\x1b\[([0-9;]*)m/g;
        let match;

        while ((match = regex.exec(cleanText)) !== null) {
            if (match.index > lastIndex) {
                html += escapeHTML(cleanText.slice(lastIndex, match.index), classes);
            }

            classes = parseSGR(match[1], classes);
            lastIndex = match.index + match[0].length;
        }

        if (lastIndex < cleanText.length) {
            html += escapeHTML(cleanText.slice(lastIndex), classes);
        }

        return { html, classes };
    }

    function escapeHTML(text, classes) {
        const escaped = text
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;');

        if (classes.length === 0) {
            return escaped;
        }
        return `<span class="${classes.join(' ')}">${escaped}</span>`;
    }

    // Extraer ruta del prompt sin retroceso catastrófico
    function extractPath(text) {
        if (!text) return null;
        const plain = stripAnsi(text);

        // Prompt de Windows cmd: reconoce "C:\>", "C:\path>", "andres@HP245G7 C:\>", etc.
        const winMatch = plain.match(/([a-zA-Z]:[\\/][^>\r\n*?"<>|]*)\s*>/);
        if (winMatch && winMatch[1]) {
            return winMatch[1].trim();
        }

        // Prompt de PowerShell: "PS C:\> " o "PS C:\path> "
        const psMatch = plain.match(/PS\s+([a-zA-Z]:[\\/][^>\r\n*?"<>|]*)\s*>/);
        if (psMatch && psMatch[1]) {
            return psMatch[1].trim();
        }

        // Prompt de Unix/Linux: "usuario@servidor:/ruta$" o "usuario@servidor:~$ "
        const unixMatch = plain.match(/(?:[a-zA-Z0-9_.-]+@[a-zA-Z0-9_.-]+:)([^$#\r\n\s]+)\s*[$#]/);
        if (unixMatch && unixMatch[1]) {
            return unixMatch[1].trim();
        }

        return null;
    }

    return {
        stripAnsi,
        hasClearScreen,
        removeClearScreen,
        renderToHTML,
        extractPath,
    };
})();
