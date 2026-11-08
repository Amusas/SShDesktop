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

    // Comandos de sistema operativo (títulos de ventana: \x1b]0;titulo\x07)
    const OSC_REGEX = /\x1b\][^\x07\x1b\r\n]*(?:\x07|\x1b\\)?/g;

    // Secuencias de control de dispositivo (DCS: \x1bP...\x07 o \x1bP...\x1b\)
    const DCS_REGEX = /\x1bP[^\x07\x1b\r\n]*(?:\x07|\x1b\\)?/g;

    // Secuencias de control CSI que no sean estilo/color (movimiento de cursor, borrado de línea, etc.)
    // Incluye modificadores privados (?, >, =) y cualquier caracter final entre @ y ~ excepto 'm'
    const NON_SGR_CSI_REGEX = /\x1b\[[\x30-\x3f]*[\x20-\x2f]*[@-ln-~]/g;

    // Secuencias de escape de 2 bytes, modos de teclado (DECPNM \x1b>, DECKPAM \x1b=), selección de fuentes, etc.
    const OTHER_ESCAPES_REGEX = /\x1b(?:[\^_X][^\x07\x1b\r\n]*(?:\x07|\x1b\\)?|[()#%*+\-\/][@-~]|[0-9<=>A-Za-z])/g;

    // Patrón global para remover todas las secuencias ANSI (para cálculos de texto plano)
    const ALL_ANSI_REGEX = /\x1b(?:\][^\x07\x1b\r\n]*(?:\x07|\x1b\\)?|P[^\x07\x1b\r\n]*(?:\x07|\x1b\\)?|\[[\x30-\x3f]*[\x20-\x2f]*[@-~]|[\^_X][^\x07\x1b\r\n]*(?:\x07|\x1b\\)?|[()#%*+\-\/][@-~]|[0-9<=>A-Za-z])/g;

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
        let s = text.replace(ALL_ANSI_REGEX, '').replace(/[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]/g, '');
        s = s.replace(/^[a-zA-Z0-9_.-]+@[a-zA-Z0-9_.-]+:[^\r\n$#>]*\s+(?=[a-zA-Z0-9_.-]+@[a-zA-Z0-9_.-]+:)/gm, '');
        s = s.replace(/(?:^|\r|\n)0;[a-zA-Z0-9_.-]+@[a-zA-Z0-9_.-]+:[^\r\n]*(?:\x07|\r|\n|$)/g, '');
        s = s.replace(/0;[a-zA-Z0-9_.-]+@[a-zA-Z0-9_.-]+:[^$#\r\n]*(?=[a-zA-Z0-9_.-]+@)/g, '');
        s = s.replace(/^0;[a-zA-Z0-9_.-]+@[^:\r\n]+:[^\r\n~]*~/g, '');
        s = s.replace(/\[\d+d/g, ''); // Limpiar artefactos residuales de cursor VPA si ESC se fragmentó
        return s;
    }

    // Detectar secuencias ANSI incompletas al final de un fragmento de stream
    const INCOMPLETE_ANSI_REGEX = /(?:\x1b\[\d*X|\x1b\[(?:1;1)?H(?:\x1b\[\?25[hl])*P?|\x1b(?:\[[\x30-\x3f]*[\x20-\x2f]*|\][^\x07\x1b\r\n]*|P[^\x07\x1b\r\n]*|[\^_X][^\x07\x1b\r\n]*|[()#%*+\-\/])?)$/;

    function splitIncompleteAnsi(text) {
        if (!text) return { complete: '', remainder: '' };
        const match = text.match(INCOMPLETE_ANSI_REGEX);
        if (match && match[0]) {
            return {
                complete: text.slice(0, match.index),
                remainder: match[0],
            };
        }
        return { complete: text, remainder: '' };
    }

    // Pre-procesa el flujo bruto antes de separar por \r y \n para evitar que secuencias
    // de cursor de ConPTY (\r\x1b[?25l\n, \x1b[1B, \x1b[4C) rompan saltos de línea o borren líneas
    function prepareStreamText(text) {
        if (!text) return '';
        let s = text
            .replace(OSC_REGEX, '')
            .replace(DCS_REGEX, '');

        // Convertir posicionamiento de cursor a columna 1 (\x1b[<fila>;1H) en salto de línea
        // 1. Eliminar artefacto de ConPTY/PSReadLine: \x1b[HP justo antes de posicionar cursor o salto
        s = s.replace(/\x1b\[(?:1;1)?H(?:\x1b\[\?25[hl])*P(?=\x1b\[|\r|\n)/g, '');

        // 2. Eliminar secuencias de borrado de caracteres en blanco de ConPTY (\x1b[<N>X\x1b[<N>C)
        // antes de convertir \x1b[<N>C independientes en espacios
        s = s.replace(/\x1b\[\d*X(?:\x1b\[\d*C)?/g, '');

        // 3. Eliminar sondeo de 1 carácter con backspace de ConPTY justo antes de \x1b[H (ej. "H \b\x1b[?25h\x1b[?25l\x1b[H")
        s = s.replace(/[^\r\n\x1b] \x08(?=(?:\x1b\[\?25[hl])*\x1b\[(?:1;1)?[Hf])/g, '');

        // 4. Marcar reinicio de cursor a inicio de pantalla de ConPTY (\x1b[H o \x1b[1;1H) con \x02
        // para deduplicar repintados completos del buffer de pantalla
        s = s.replace(/\x1b\[(?:1;1)?[Hf]/g, '\x02');

        s = s.replace(/\x1b\[\d+;1[Hf]/g, '\n');

        // Convertir movimiento de cursor hacia abajo / siguiente línea (\x1b[B, \x1b[1B, \x1b[1E) en salto de línea
        s = s.replace(/\x1b\[(\d*)[BE]/g, (_, n) => {
            const count = Math.min(Math.max(parseInt(n || '1', 10), 1), 5);
            return '\n'.repeat(count);
        });

        // Convertir avance horizontal de cursor (\x1b[nC) en espacios para preservar columnas en dir/tablas
        s = s.replace(/\x1b\[(\d*)C/g, (_, n) => {
            const count = Math.min(Math.max(parseInt(n || '1', 10), 1), 40);
            return ' '.repeat(count);
        });

        // Eliminar el resto de secuencias CSI que no son de color/estilo (como \x1b[?25l, \x1b[K, etc.)
        s = s
            .replace(NON_SGR_CSI_REGEX, '')
            .replace(OTHER_ESCAPES_REGEX, '')
            .replace(/[\x00\x01\x03-\x07\x0b\x0e-\x1a\x1c-\x1f\x7f]/g, '');

        return s;
    }

    // Detectar secuencias ANSI de borrado total de pantalla (cls / clear)
    // Nota: No incluir \x1b[J ni \x1b[0J porque ConPTY y bash los usan para limpiar debajo del cursor sin borrar la pantalla
    function hasClearScreen(text) {
        if (!text) return false;
        return text.includes('\x1b[2J') ||
               text.includes('\x1b[3J') ||
               text.includes('\x1bc') ||
               text.includes('\x0c') ||
               text.includes('\f');
    }

    function removeClearScreen(text) {
        if (!text) return '';
        return text
            .replace(/\x1b\[[23][Jj]/g, '')
            .replace(/\x1bc/g, '')
            .replace(/\x1b\[(?:H|1;1H|2;0H|[0-9;]*[fH])/g, '')
            .replace(/[\x0c\f]/g, '');
    }

    function sanitizeText(text) {
        if (!text) return '';
        let clean = text
            .replace(OSC_REGEX, '')
            .replace(DCS_REGEX, '')
            .replace(NON_SGR_CSI_REGEX, '')
            .replace(OTHER_ESCAPES_REGEX, '')
            .replace(/[\x00-\x07\x0b\x0e-\x1a\x1c-\x1f\x7f]/g, '');

        // Limpieza de títulos de terminal filtrados de OpenSSH o fragmentos de prompt duplicados
        clean = clean.replace(/^[a-zA-Z0-9_.-]+@[a-zA-Z0-9_.-]+:[^\r\n$#>]*\s+(?=[a-zA-Z0-9_.-]+@[a-zA-Z0-9_.-]+:)/gm, '');
        clean = clean.replace(/(?:^|\r|\n)0;[a-zA-Z0-9_.-]+@[a-zA-Z0-9_.-]+:[^\r\n]*(?:\x07|\r|\n|$)/g, '');
        clean = clean.replace(/0;[a-zA-Z0-9_.-]+@[a-zA-Z0-9_.-]+:[^$#\r\n]*(?=[a-zA-Z0-9_.-]+@)/g, '');
        clean = clean.replace(/^0;[a-zA-Z0-9_.-]+@[^:\r\n]+:[^\r\n~]*~/g, '');

        // Limpieza de artefactos residuales de ConPTY (letras 'P' y escapes fragmentados)
        clean = clean.replace(/\[\d+d/g, ''); // Artefactos de cursor como [103d
        clean = clean.replace(/PPS\s+([A-Za-z]:[\\/])/g, 'PS $1');
        clean = clean.replace(/^P(?=PS\s|[a-zA-Z]:[\\/]|dir|ls|cd)/gm, '');
        clean = clean.replace(/\bP(dir|cd|ls|cls|clear|echo)\b/g, '$1');
        clean = clean.replace(/^>>\s*$/gm, '');
        clean = clean.replace(/^>\s*$/gm, '');
        clean = clean.replace(/^[>P\s]+$/gm, '');

        // Cualquier caracter ESC solitario restante que no pertenecía a un SGR válido
        clean = clean.replace(/\x1b(?!\[[0-9;]*m)/g, '');

        return clean;
    }

    // Formatear líneas de resultados que contienen rutas de directorio (CMD y PowerShell)
    // Ejemplo: "    Directorio: C:\Users\andres.HP245G7" o " Directorio de C:\Users\andres.HP245G7"
    function formatDirectoryResultLine(text) {
        if (!text) return null;
        const plain = stripAnsi(text);

        // PowerShell: "    Directorio: C:\Users\..." o "Directorio: C:\..." o "Directory: C:\..."
        const psDirMatch = plain.match(/^(\s*(?:Directorio|Directory):\s*)([a-zA-Z]:[\\/][^\r\n]*|\S+.*)$/i);
        if (psDirMatch) {
            const prefix = psDirMatch[1];
            const pathPart = psDirMatch[2].trim();
            return `<span class="result-dir-label">${escapeHTML(prefix, [])}</span><span class="result-path-highlight">${escapeHTML(pathPart, [])}</span>`;
        }

        // Windows CMD: " Directorio de C:\Users\..." o "Directory of C:\..."
        const cmdDirMatch = plain.match(/^(\s*(?:Directorio de|Directory of)\s*)([a-zA-Z]:[\\/][^\r\n]*|\S+.*)$/i);
        if (cmdDirMatch) {
            const prefix = cmdDirMatch[1];
            const pathPart = cmdDirMatch[2].trim();
            return `<span class="result-dir-label">${escapeHTML(prefix, [])}</span><span class="result-path-highlight">${escapeHTML(pathPart, [])}</span>`;
        }

        return null;
    }

    // Intenta formatear una línea que contiene prompt y/o comando separando usuario, ruta y comando
    function formatPromptLine(text) {
        if (!text) return null;
        const plain = stripAnsi(text);

        // 1. Windows CMD: "hp 245 ryzen 3@HP245G7 C:\path>command" o "C:\path>command"
        const cmdMatch = plain.match(/^(\s*)(?:([^\r\n>]+?)\s+)?([a-zA-Z]:[\\/][^>:\r\n*?"<>|]*)(>)\s*(.*)$/);
        if (cmdMatch) {
            const indent = cmdMatch[1] || '';
            const userPart = cmdMatch[2] ? cmdMatch[2].trim() : '';
            const pathPart = cmdMatch[3].trim();
            const charPart = cmdMatch[4];
            const cmdPart = cmdMatch[5] ? cmdMatch[5].trim() : '';

            let html = indent;
            if (userPart) {
                html += `<span class="prompt-user">${escapeHTML(userPart, [])}</span> `;
            }
            html += `<span class="prompt-path-highlight">${escapeHTML(pathPart, [])}</span>`;
            html += `<span class="prompt-char">${escapeHTML(charPart, [])}</span>`;

            if (cmdPart) {
                html += ` <span class="line-command">${escapeHTML(cmdPart, [])}</span>`;
            }
            return html;
        }

        // 2. PowerShell: "PS C:\path> command" o "PS C:\path>"
        const psMatch = plain.match(/^(\s*)(PS\s+)([a-zA-Z]:[\\/][^>:\r\n*?"<>|]*)(>)\s*(.*)$/);
        if (psMatch) {
            const indent = psMatch[1] || '';
            const psLabel = psMatch[2];
            const pathPart = psMatch[3].trim();
            const charPart = psMatch[4];
            const cmdPart = psMatch[5] ? psMatch[5].trim() : '';

            let html = indent;
            html += `<span class="prompt-ps">${escapeHTML(psLabel, [])}</span>`;
            html += `<span class="prompt-path-highlight">${escapeHTML(pathPart, [])}</span>`;
            html += `<span class="prompt-char">${escapeHTML(charPart, [])}</span>`;

            if (cmdPart) {
                html += ` <span class="line-command">${escapeHTML(cmdPart, [])}</span>`;
            }
            return html;
        }

        // 3. Unix / Linux / Docker: "user@host:~$ command" o "root@container:/var/log# command"
        const unixMatch = plain.match(/^(\s*)(?:[a-zA-Z0-9_.-]+@[a-zA-Z0-9_.-]+:[^\r\n$#>]*\s+)?([a-zA-Z0-9_.-]+@[a-zA-Z0-9_.-]+)(:)([^:$#\r\n\s]+)([$#])\s*(.*)$/);
        if (unixMatch) {
            const indent = unixMatch[1] || '';
            const userHost = unixMatch[2];
            const sep = unixMatch[3];
            const pathPart = unixMatch[4];
            const charPart = unixMatch[5];
            const cmdPart = unixMatch[6] ? unixMatch[6].trim() : '';

            let html = indent;
            html += `<span class="prompt-user">${escapeHTML(userHost, [])}</span>`;
            html += `<span class="prompt-sep">${escapeHTML(sep, [])}</span>`;
            html += `<span class="prompt-path-highlight">${escapeHTML(pathPart, [])}</span>`;
            html += `<span class="prompt-char">${escapeHTML(charPart, [])}</span>`;

            if (cmdPart) {
                html += ` <span class="line-command">${escapeHTML(cmdPart, [])}</span>`;
            }
            return html;
        }

        // 4. Contenedores Alpine / BusyBox / sh: "/ # command", "/app $ command", "sh-5.1# command"
        const shMatch = plain.match(/^(\s*)(\/[^\s$#\r\n]*|(?:sh|bash|ash|dash)-[0-9.]+)\s*([$#])\s*(.*)$/);
        if (shMatch) {
            const indent = shMatch[1] || '';
            const pathOrShell = shMatch[2];
            const charPart = shMatch[3];
            const cmdPart = shMatch[4] ? shMatch[4].trim() : '';

            let html = indent;
            html += `<span class="prompt-path-highlight">${escapeHTML(pathOrShell, [])}</span> `;
            html += `<span class="prompt-char">${escapeHTML(charPart, [])}</span>`;

            if (cmdPart) {
                html += ` <span class="line-command">${escapeHTML(cmdPart, [])}</span>`;
            }
            return html;
        }

        return null;
    }

    function renderPromptAndCommand(promptText, commandText) {
        const p = promptText ? promptText.trim() : '';
        const c = commandText ? commandText.trim() : '';
        const combined = p + (c ? ' ' + c : '');
        const formatted = formatPromptLine(combined);
        if (formatted) return formatted;

        let html = '';
        if (p) {
            html += `<span class="line-prompt">${escapeHTML(p, [])}</span>`;
        }
        if (c) {
            html += (p ? ' ' : '') + `<span class="line-command">${escapeHTML(c, [])}</span>`;
        }
        return html;
    }

    function renderToHTML(text, activeClasses) {
        // 1. Verificar si la línea completa corresponde a una línea de prompt/comando
        const promptFormatted = formatPromptLine(text);
        if (promptFormatted) {
            return { html: promptFormatted, classes: [] };
        }

        // 2. Verificar si es una línea de resultado con ruta de directorio (Directorio: o Directorio de)
        const dirFormatted = formatDirectoryResultLine(text);
        if (dirFormatted) {
            return { html: dirFormatted, classes: [] };
        }

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
        if (!text) return '';
        const escaped = text
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;');

        if (!classes || classes.length === 0) {
            return escaped;
        }
        return `<span class="${classes.join(' ')}">${escaped}</span>`;
    }

    // Extraer ruta del prompt de forma precisa
    function extractPath(text) {
        if (!text) return null;
        const plain = stripAnsi(text);

        // Prompt de PowerShell: "PS C:\path> "
        const psMatch = plain.match(/PS\s+([a-zA-Z]:[\\/][^>:\r\n*?"<>|]*)\s*>/);
        if (psMatch && psMatch[1]) {
            return psMatch[1].trim();
        }

        // Prompt de Windows cmd: "C:\path>" o "usuario@host C:\path>"
        const winMatch = plain.match(/([a-zA-Z]:[\\/][^>:\r\n*?"<>|]*)\s*>/);
        if (winMatch && winMatch[1]) {
            return winMatch[1].trim();
        }

        // Prompt de Unix/Linux/Docker: "usuario@servidor:/ruta$" o "root@contenedor:/ruta# "
        const unixMatch = plain.match(/(?:[a-zA-Z0-9_.-]+@[a-zA-Z0-9_.-]+:)([^$#\r\n\s]+)\s*[$#]/);
        if (unixMatch && unixMatch[1]) {
            return unixMatch[1].trim();
        }

        // Prompt de contenedor sh/Alpine: "/ruta # " o "/ # "
        const shPathMatch = plain.match(/(?:^|\n)\s*(\/[^\s$#\r\n]*)\s+[$#]\s*$/);
        if (shPathMatch && shPathMatch[1]) {
            return shPathMatch[1].trim();
        }

        return null;
    }

    return {
        stripAnsi,
        splitIncompleteAnsi,
        prepareStreamText,
        hasClearScreen,
        removeClearScreen,
        sanitizeText,
        renderToHTML,
        renderPromptAndCommand,
        formatPromptLine,
        formatDirectoryResultLine,
        extractPath,
    };
})();
