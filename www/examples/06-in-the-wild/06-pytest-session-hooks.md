<!-- Generated from 06-in-the-wild/06-pytest-session-hooks.dgm by `bazel run //site:sync`. Do not edit: //site:site_test fails while this file disagrees with its source, and the next sync overwrites it. -->

# pytest tests/ — one session, hook by hook

One pytest run, hook by hook. The diagram is the `stateDiagram-v2` that BeniaminK posted in pytest-dev/pytest#3261 (https://github.com/pytest-dev/pytest/issues/3261), unchanged — the thread has asked since 2018 for "a flowchart of all the pytest test session states together with all the applicable hooks", and the diagram answers the *which* but not the *when*. The scenario adds the when: every step is one phase of the session, lit, with everything the run has not reached yet greyed back, and a note saying what a plugin author can do at that hook.

<div class="cinegram" data-cinegram="06-in-the-wild/06-pytest-session-hooks" data-height="1080"></div>

[Edit in the playground](../../playground/#doc=vFrhjhu3EX6VwRbG6Q4r3Z2dBIGMAE2dtgiapCniP4VlnKjlSMscl9ySXMmqYaC_-gBF36X_-yh5kmJIrnal5erkJq1_nGTucEgO55uZb1bvs202v88zVM7ss3l299lUqKkrcboTkt_efTat9w6tm1q0Vmg1LbV-tDO-qbI80zWqj52zFhJtNn_zPqs_dqrL5pnDdy7LM57Ns2fP4I8KIUwA06gcSBpWe_85g9clAhdsY1gFwoIrEZbWMYdfhcHp9vkSXMncQj17Br9BJVgl1B-g1tYhB6Gi7inH7W34-qsXzz-79-KT0rnazm9vN8KVzWpW6Op2IH4rrG3Q3tKs6xwaVZRMbZDDT3_7p9-PKw0y7hWWzAKzj8jBClUgPL-7_xzW2sAiY7CWeleUzDjQa2BS-snx5P5PtBf481mv0OkNuhIN7IQrD5NYXUtRsJVEbyW7yHJgivtnrbGYsjs03mJe082uFEV5A6vGgdLOy97sSlQ3wci2QMWM0MA4D3amh3PALZo9WIc12V-roK0umUU6BwnGfecghcvDTv0sVwq18RKmUd42tLBBVpTIYY_hzjYG98hhxYrHcApGYgiW7Wn6rmQOGNSy2QgFrHGlNlAwBVwDc_7qg6t4ZdPp1H--0lWtrSA13pgwWb7SUmLhhFbLHJZfCYOF02Z_PPx7VK_ROru8BmbCUZkSFaPn4JjZoLMgxSMCU3vQ_mr8CjlYDUsuKugp9GfrrmBXaolQHJ5HK3qDaYWw04YHEyzJV6JvPETzWkeuM100d3cvMB6yWynsEsPNOcOUFWHPZB9UjlxBuBks39y8XdJNMs4NWktOFI5pYWm0dg9-nSXcxv-i4sscvPPQtFLv_BJrYazze5WMvhijdxYqvcXji3h92IoFJgkoe6iNUN4BhQG9UyDZCiVd0f1s9t3y2puSDGChRINQMGP2XpkiKxHsWJgDO91IDtywHTCwWGjFodD1Hujkup4t1EmsWCgACDcGiywamHHuYTQJTlYxxTZorhcZGeVEZkyBrumMk5oZiyaH85qCcFJVmPhgcCOsQ4M8bqpV-aBYhTkkNQ_mJleQmvEHoYQTTD4UWq1p1E6QGbn3_xebHNpzMLOxx4ukpydXip7-wFusTWrmSq8clTtWO5BNahQbpQ22wpMOSg-tZvobDnGs_3jm2e1SdpvQnyOl5zZNwunb3LciFXvESvNG4iR8PKk8NfeCRYTDqjWMNjkEf9GrH89pp0nnbEIWrjQX6z1J2skh4rf-4oeTxjmZmlwlqlsLJWzZKcd3wpFkc6L5SDwobNesuBQKHyom1Dw1OGl942iWH2sMzgcjSfl-QJ6nBtsjnC7TGmQ-HEpP2aBCwxw-eJDN08OTCh1bN6oYW-4hWGo--iS9eKOGhumNdaYJs97cvIVpTE_pqHkymJLuR8bT0YH8SLwbewoDBedi2VmRgaa-j4165HBWa8u0Ow7k-z426pHdrK4-6KOuG4X3YTzpNJ2aRKkEc6Bc3U0PqlOSvTWG63ThfnDW44A9WPCjdPXj9CWavNzA2xLxeKDsQ4sG-pcyx0FrW2keVJwa8yCQtuAwgA92fBwmRjZ5WGbEaMn4nQ4zPbFz2vpx-7CVlImeDPdHTwbzerFqLK51c97cvKWtHFhQm5s8M7O3vuqk6vOQnPo0dZHBe7A1IqfLvGtP5UlTCEMWFtn38VsbkWCFa208kwhMSVg47IyTzrBtjraYwyJ7fSi7iXFG6uep1a4kh40bFhasE1ICsxarlfSanUW5ng0isCQ6cyBXHAvJDILCXTierbGwL4eBWFgih9581X66lmzjmYBQAh6J8JASxjlycDqwXF9OvhwPzWthkNhlgVCjgbbmFaqQDW9Z5KoR0k2FsjP4ju4i0M_Ks0GLCCzQ6D262SKLthPV-UiepwJ1PgjFeSrS5j2vzZM-mY_6osW_dMD2bK8jXzCSReE98MbM4ZO7u8qSk_VmD3LrSGq9TEebcJ_It0ll8aMUm1KKTelOt5YP1snPLWDdXuIcWOHEFlvlvkEw2O4iC3426_jY1XRqpd5dXS8WqnsolJhcOVGhbtxVDrPZ7DpAWHCcg_G79iv1gdy6DCyyr4MXQVcPkHPl5I3Ke2qhq8pzY6HQk-3CIQethqA-X2aIqtbG2agzjM7qPfgGnCe5JXrH4cJ4CGoVGzbEK2CvG6gpDIRmlXBtC81z9kpXSDxcQ-hoQZ8CzpIFjGg7Q1qGpg7zLaEQLSaHQEKANOgao4ApX8ZDoTkGOk_bVDpEPJQWoWR1jcpevxwWQIdAU2nrDqGU69g8sOiaOocVFqyxGIMnTVoLxWQqCPzv4DvqwhfVnAckfToKy7FKdLwQvUDpcXk6Up0m9YzB_GeF2Uvw3gkvsuirAdJbJht8IO2Tq4qZRzS2BfdCHZJu60S-MaYQeXSbpwNA254NmbjXrCWEMt95OtQwK6RFRuF-VK9HUIXU7pEUmr3wQ1xCr37EwnUtXo9egl0sIGLilz7wMxe1SbR767AK7d0A2VenvUeKTSvd-ChgGvUyxggpHEGbwUrq4jHg1oZUzhoukPK0RQztwKopyrYV3GvVX62FCqmbHOAKtmhsY-HKNErFjF5dncHofw3BXnU3Tp8uwEaq73rEnoKGTy9JfidLJ528m3RM3Po-yIWhUrJlFgKD5_nbzj2RAMprFpAV5aj79Rz1URSPFvR6HTrQTD7O4HfaxCzSUapRriXiG498hLhRLtii0-RZE1cK2wV1kisaY9Gfis4xnYbZZJnr4PB0ELCN2YoteY2nZhvtG91GN5syxd1ewvd7V2oV8-QKC11RefhtIG1bwc6RugAYb8uYYV2viR3QgYbJOT2zbR1OFWyvgGWuPPLtlmbl53lTPsqX8l8OHh3TOsPwL8gdQ959lsJHjS9-hsYjgn65vsDnL6DzKZUDdCasFUQ2rNmMbdW_ryAIepdcZDn4dEUjkWH--19w_xxm9X6RpdLeiTEXWSywXpsG4ae__wPso6jBA4ycb7FQkyN85Qdw5YdS8oKqN7D5RfZbgqFPM-tGtTmOYEXA9VIvieWxCp0Rf0XAdzVTPLzBGcahrxUtGoNMsH1-prFBB6VsFB8g911tEMppWv1rkploA6yNnNpcz9JdkDbbBvqOpVAcfh0kZ1Q0zPqHiEWzaVSPmx4bIabqtgXrQ1bF9kR_oWBS2hm8aqzT1XHh0V6j0moaY1XY3uTPX377TQ4__Omba5CUHISKYcbtdAg1qZz5_48myebWaW_r6SCSamOd62JdRjnbDSRwOVB3QGbob_WR-fknHln3L-6BrX3x2DlHi9I-WILpY4HozyJF6MishfRsIAeD2vDwlXzFIpOpZs_57prBAsU2vu1dN1KGhVrnqxrfQaRaTkEtWRFcefoYkmxFq-CB-RimuK7Ab0uoTQ7vuLDuygJ9GLFq_N0yKSNT85iejbpWizCmlG5UgYFklrEu7QxCDA2-9GalqJW3PxTw-Ao5daK0CwAQti0trwHfYdE4DJUB2eMXLSK7fugl7dDL0-SgQfpEf_SjSNfHwv8ynpXcd8TJm_lb-ALeCP_rFkGXFB6LNTGut4tFtPTM_zYjqmzdDvnES39xWecl7vmYdyFlF3LnQ1xFZoDrnXqKdUV1nr-Qz_n-RHjPGNqYbX1qMHRfmEHYGeEcKpj82Cjh3lWSsugWDdtgl2r6neWIgkoYow3RowE7aX8_5Omcz0eRRNVGF2jtDL7vpwxdo0IOBq1uTIGWbN4pK6SmHFFiFeDZQmLo4Il4nbRPit4kCVLbfh_tvl8AkqOefFTU_vzko7qMT-J-zPUtujE7rBjf-HzQvZCGu178zz68_fCfAAAA__8){ .md-button }

??? abstract "The source — `06-in-the-wild/06-pytest-session-hooks.dgm`"

    ```dgm
    %% One pytest run, hook by hook. The diagram is the `stateDiagram-v2` that
    %% BeniaminK posted in pytest-dev/pytest#3261
    %% (https://github.com/pytest-dev/pytest/issues/3261), unchanged — the thread
    %% has asked since 2018 for "a flowchart of all the pytest test session states
    %% together with all the applicable hooks", and the diagram answers the
    %% *which* but not the *when*. The scenario adds the when: every step is one
    %% phase of the session, lit, with everything the run has not reached yet
    %% greyed back, and a note saying what a plugin author can do at that hook.
    %% ---
    %% Composite states (`Collection`, `DirectoryCollection`, `GenTests`) are
    %% animation targets like any other state, so `dim Collection` greys the
    %% whole collection phase with one word, and `flow pytest_sessionstart ->
    %% Collection` animates the transition that enters it. `[*]` is addressable
    %% as `root_start` / `root_end`, which is how the first and last arrows move.
    %% ---
    %% Transitions already print their own labels (`1..N`), so flows here carry
    %% none — a label would draw a second copy on top.
    stateDiagram-v2
        state "pytest_addhooks(pluginmanager)" as pytest_addhooks
        state "pytest_addoption(parser, pluginmanager)" as pytest_addoption
        state "pytest_plugin_registered(plugin, plugin_name, manager)" as pytest_plugin_registered
        state "pytest_load_initial_conftests(early_config, parser, args)" as pytest_load_initial_conftests
        state "pytest_collect_directory(path, parent)" as pytest_collect_directory
        state "pytest_ignore_collect(collection_path, path, config)" as pytest_ignore_collect
        state "pytest_collect_file(file_path, path, parent)" as pytest_collect_file
        state "pytest_pycollect_makemodule(module_path, path, parent)" as pytest_pycollect_makemodule
        state "pytest_pycollect_makeitem(collector, name, obj)" as pytest_pycollect_makeitem
        state "pytest_collection_modifyitems(session, config, items)" as pytest_collection_modifyitems
        state "pytest_sessionfinish(session, exitstatus)" as pytest_sessionfinish
        pytest_cmdline_main: pytest_cmdline_main(config)
        pytest_configure: pytest_configure(config)
        pytest_sessionstart: pytest_sessionstart(session)
        pytest_collection: pytest_collection(session)
        pytest_generate_tests: pytest_generate_tests(metafunc)
        pytest_collection_finish: pytest_collection_finish(session)
        pytest_unconfigure: pytest_unconfigure(config)

        [*] --> pytest_addhooks
        pytest_addhooks --> pytest_addoption
        pytest_addoption --> pytest_plugin_registered
        pytest_plugin_registered  --> pytest_load_initial_conftests
        pytest_load_initial_conftests --> pytest_cmdline_main
        pytest_cmdline_main --> pytest_configure
        pytest_configure --> pytest_sessionstart
        pytest_sessionstart --> Collection
        state Collection {
            pytest_collection --> DirectoryCollection : 1..N
            state DirectoryCollection {
                pytest_collect_directory --> pytest_ignore_collect : 1..N
                pytest_collect_directory --> pytest_collect_file : 1..N
                pytest_collect_file --> pytest_pycollect_makemodule : 1..N
            }

            DirectoryCollection --> GenTests : 1..N

            state GenTests {
                pytest_pycollect_makeitem --> pytest_generate_tests
            }

            GenTests --> pytest_collection_modifyitems
            pytest_collection_modifyitems --> pytest_collection_finish
        }

        Collection --> pytest_sessionfinish
        pytest_sessionfinish --> pytest_unconfigure
        pytest_unconfigure --> [*]

    scenario "pytest tests/ — one session, hook by hook" { speed: 1.0 }

      step plugins "Plugins register before anything is configured" {
        desc: "The first three hooks run while pytest is still assembling itself. pytest_addhooks lets a plugin declare new hook specs; pytest_addoption is where --my-flag and ini keys are added to the parser; pytest_plugin_registered fires once per plugin, including the built-ins. None of them can see a test yet."
        dim pytest_load_initial_conftests, pytest_cmdline_main, pytest_configure, pytest_sessionstart, Collection, pytest_sessionfinish, pytest_unconfigure
        seq {
          flow root_start -> pytest_addhooks { dur: 400ms }
          flow pytest_addhooks -> pytest_addoption { dur: 400ms }
          flow pytest_addoption -> pytest_plugin_registered { dur: 400ms }
        }
        highlight pytest_addhooks, pytest_addoption, pytest_plugin_registered { style: active }
        note pytest_addoption "parser.addoption('--slow')\nparser.addini('timeout', ...)" { side: right }
      }

      step conftest "Initial conftests load, then the command line is acted on" {
        desc: "pytest_load_initial_conftests imports the conftest.py files on the rootdir and on every path you passed — it is the last moment to change early_config. pytest_cmdline_main is the whole run as one hook (a plugin can return an exit code here and nothing else happens); pytest_configure is where most plugins do their setup, because config is final."
        dim pytest_sessionstart, Collection, pytest_sessionfinish, pytest_unconfigure
        seq {
          flow pytest_plugin_registered -> pytest_load_initial_conftests { dur: 450ms }
          flow pytest_load_initial_conftests -> pytest_cmdline_main { dur: 450ms }
          flow pytest_cmdline_main -> pytest_configure { dur: 450ms }
        }
        highlight pytest_load_initial_conftests, pytest_cmdline_main, pytest_configure { style: active }
        note pytest_configure "config.addinivalue_line('markers', ...)\nregister plugins that need config" { side: right }
      }

      step session "The session starts and collection begins" {
        desc: "pytest_sessionstart is the first hook with a Session object, and the last one before pytest looks at the filesystem. The whole Collection phase is about to run; it is lit as a block here so the audience sees how much of the diagram is 'finding tests' versus 'running them'."
        dim pytest_sessionfinish, pytest_unconfigure
        seq {
          flow pytest_configure -> pytest_sessionstart { dur: 450ms }
          flow pytest_sessionstart -> Collection { dur: 500ms }
        }
        highlight pytest_sessionstart { style: active }
        highlight Collection
      }

      step dirs "Directories and files, 1..N times each" {
        desc: "pytest_collection kicks off one walk. For every directory pytest_collect_directory is asked, pytest_ignore_collect can veto it (this is where norecursedirs and --ignore act), and each surviving file goes through pytest_collect_file; Python files become a Module via pytest_pycollect_makemodule. The 1..N on the transitions is literal: these hooks fire once per path."
        dim GenTests, pytest_collection_modifyitems, pytest_collection_finish, pytest_sessionfinish, pytest_unconfigure
        seq {
          flow pytest_collection -> DirectoryCollection { dur: 450ms }
          flow pytest_collect_directory -> pytest_ignore_collect { dur: 350ms }
          flow pytest_collect_directory -> pytest_collect_file { dur: 350ms }
          flow pytest_collect_file -> pytest_pycollect_makemodule { dur: 350ms }
        }
        highlight DirectoryCollection
        gauge pytest_collect_file { label: "files", value: "tests/ · 12 .py" }
        note pytest_ignore_collect "return True → skip this path\n(norecursedirs, --ignore, conftest)" { side: right }
      }

      step items "Each test function becomes an item; parametrize expands here" {
        desc: "Inside every module, pytest_pycollect_makeitem turns a collected name into an Item (or a Collector). pytest_generate_tests is the hook behind @pytest.mark.parametrize — it runs once per test function with a metafunc and may add calls. Custom plugins that collect non-Python tests (YAML, SQL) live in these two hooks."
        dim pytest_collection_modifyitems, pytest_collection_finish, pytest_sessionfinish, pytest_unconfigure
        seq {
          flow DirectoryCollection -> GenTests { dur: 450ms }
          flow pytest_pycollect_makeitem -> pytest_generate_tests { dur: 400ms }
        }
        highlight GenTests
        gauge pytest_generate_tests { label: "items", value: "84 → 131 after parametrize" }
      }

      step modify "The item list is filtered, reordered, and sealed" {
        desc: "pytest_collection_modifyitems receives the full list and may mutate it in place — -k and -m deselection, random ordering, xdist's distribution all happen here. pytest_collection_finish is the announcement that the list is final. After this, the runtest hooks (not in this diagram) execute each item."
        dim pytest_sessionfinish, pytest_unconfigure
        seq {
          flow GenTests -> pytest_collection_modifyitems { dur: 450ms }
          flow pytest_collection_modifyitems -> pytest_collection_finish { dur: 450ms }
        }
        highlight pytest_collection_modifyitems, pytest_collection_finish { style: active }
        note pytest_collection_modifyitems "items[:] = [i for i in items if ...]\nconfig.hook.pytest_deselected(items=...)" { side: right }
      }

      step finish "The session ends and plugins tear down" {
        desc: "pytest_sessionfinish sees the exit status and is where reports are written (junitxml, coverage). pytest_unconfigure is the mirror of pytest_configure — the last hook of the process. Plugins that opened resources in configure close them here."
        seq {
          flow Collection -> pytest_sessionfinish { dur: 500ms }
          flow pytest_sessionfinish -> pytest_unconfigure { dur: 450ms }
          flow pytest_unconfigure -> root_end { dur: 400ms }
        }
        highlight pytest_sessionfinish, pytest_unconfigure { style: active }
        set pytest_sessionfinish { badge: "exitstatus 0" }
      }
    ```

← [claude code tool call](../06-in-the-wild/05-claude-code-tool-call.md)  
→ [architecture canvas](../06-in-the-wild/07-architecture-canvas.md)
