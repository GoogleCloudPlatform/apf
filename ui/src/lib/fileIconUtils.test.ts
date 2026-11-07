/**
 * Copyright 2026 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// @vitest-environment jsdom
import { describe, it, expect } from "vitest"
import { getFileIconMeta } from "./fileIconUtils"

describe("getFileIconMeta utility", () => {
  it("correctly maps top 15 language file extensions to icon metadata and styles", () => {
    // Python
    const py = getFileIconMeta("script.py")
    expect(py.language).toBe("Python")
    expect(py.iconId).toBe("python")
    expect(py.textColorClass).toBe("text-[#3776ab] dark:text-[#4584b6]")

    // Jupyter Notebook
    const ipynb = getFileIconMeta("analysis.ipynb")
    expect(ipynb.language).toBe("Jupyter Notebook")
    expect(ipynb.iconId).toBe("python-ipynb")

    // Go
    const go = getFileIconMeta("main.go")
    expect(go.language).toBe("Go")
    expect(go.iconId).toBe("go")

    // Go mod
    const goMod = getFileIconMeta("go.mod")
    expect(goMod.language).toBe("Go Module")
    expect(goMod.iconId).toBe("go-mod")

    // Java
    const java = getFileIconMeta("App.java")
    expect(java.language).toBe("Java")
    expect(java.iconId).toBe("java")

    // Markdown
    const md = getFileIconMeta("README.md")
    expect(md.language).toBe("Markdown")
    expect(md.iconId).toBe("markdown")

    // HTML
    const html = getFileIconMeta("index.html")
    expect(html.language).toBe("HTML")
    expect(html.iconId).toBe("html")

    // CSS / SCSS
    const css = getFileIconMeta("styles.css")
    expect(css.language).toBe("CSS")
    expect(css.iconId).toBe("css")

    const scss = getFileIconMeta("theme.scss")
    expect(scss.language).toBe("SCSS")
    expect(scss.iconId).toBe("css-scss")

    // JavaScript / JSX
    const js = getFileIconMeta("app.js")
    expect(js.language).toBe("JavaScript")
    expect(js.iconId).toBe("javascript")

    const jsx = getFileIconMeta("Button.jsx")
    expect(jsx.language).toBe("React JSX")
    expect(jsx.iconId).toBe("javascript-jsx")

    // TypeScript / TSX
    const ts = getFileIconMeta("utils.ts")
    expect(ts.language).toBe("TypeScript")
    expect(ts.iconId).toBe("typescript")

    const tsx = getFileIconMeta("Component.tsx")
    expect(tsx.language).toBe("React TSX")
    expect(tsx.iconId).toBe("typescript-tsx")

    // Kotlin
    const kt = getFileIconMeta("Main.kt")
    expect(kt.language).toBe("Kotlin")
    expect(kt.iconId).toBe("kotlin")

    // C#
    const cs = getFileIconMeta("Program.cs")
    expect(cs.language).toBe("C#")
    expect(cs.iconId).toBe("csharp")

    // Terraform & HCL
    const tf = getFileIconMeta("main.tf")
    expect(tf.language).toBe("Terraform")
    expect(tf.iconId).toBe("terraform")

    const hcl = getFileIconMeta("config.hcl")
    expect(hcl.language).toBe("HCL")
    expect(hcl.iconId).toBe("terraform-hcl")
  })

  it("handles exact special filenames like Dockerfile, CMakeLists.txt, .gitignore and text files", () => {
    const cmakeList = getFileIconMeta("CMakeLists.txt")
    expect(cmakeList.language).toBe("CMake")
    expect(cmakeList.iconId).toBe("cmake")

    const cmakeExt = getFileIconMeta("build.cmake")
    expect(cmakeExt.language).toBe("CMake")
    expect(cmakeExt.iconId).toBe("cmake")

    const txt = getFileIconMeta("notes.txt")
    expect(txt.language).toBe("Text Document")
    expect(txt.iconId).toBe("text")

    const docker = getFileIconMeta("Dockerfile")
    expect(docker.language).toBe("Docker")
    expect(docker.iconId).toBe("docker")

    const gitignore = getFileIconMeta(".gitignore")
    expect(gitignore.language).toBe("Git")
    expect(gitignore.iconId).toBe("git")
  })

  it("falls back gracefully for unknown extensions", () => {
    const unknown = getFileIconMeta("file.xyz_unknown")
    expect(unknown.language).toBe("File")
    expect(unknown.iconId).toBe("file")
  })
})
