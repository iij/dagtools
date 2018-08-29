========
dagtools
========

IIJGIO ストレージ＆アナリシスサービス ストレージ専用コマンドラインツール

インストール
============

ご利用の環境に該当するパッケージを以下からダウンロードしてご利用ください。

- Linux (RHEL/CentOS kernel-2.6.18-274 以降、CentOS 6.0 以降を推奨)
    - 64 bit: `dagtools-linux-amd64-1.6.0.tar.gz <https://storage-dag.iijgio.com/support/tools/dagtools/dagtools-linux-amd64-1.6.0.tar.gz>`_
    - 32 bit: `dagtools-linux-386-1.6.0.tar.gz <https://storage-dag.iijgio.com/support/tools/dagtools/dagtools-linux-386-1.6.0.tar.gz>`_
- Mac OS X (10.6 以降)
    - 64 bit: `dagtools-darwin-amd64-1.6.0.tar.gz <https://storage-dag.iijgio.com/support/tools/dagtools/dagtools-darwin-amd64-1.6.0.tar.gz>`_
    - 32 bit: `dagtools-darwin-386-1.6.0.tar.gz <https://storage-dag.iijgio.com/support/tools/dagtools/dagtools-darwin-386-1.6.0.tar.gz>`_
- Windows (XP 以降)
    - 64 bit: `dagtools-windows-amd64-1.6.0.zip <https://storage-dag.iijgio.com/support/tools/dagtools/dagtools-windows-amd64-1.6.0.zip>`_
    - 32 bit: `dagtools-windows-386-1.6.0.zip <https://storage-dag.iijgio.com/support/tools/dagtools/dagtools-windows-386-1.6.0.zip>`_

上記以外のパッケージを希望される場合は dag-support@iij.ad.jp までご連絡ください。

.. warning::

   - **CentOS 5.6 x86-64 (kernel-2.6.18-238)** の環境では Linux 64 bit版のパッケージは動作しないため、Linux 32 bit版のパッケージをご利用ください。


設定
====

| INI形式の設定ファイルを `/etc/dagtools.ini` もしくはカレントディレクトリに配置するか `-f` オプションでコマンド実行時に指定します。
| 設定パラメータは以下の通りです。

**[dagtools] セクション**

===========  ================================================================================
proxy        HTTP Proxy を指定
verbose      コマンドの実行内容を表示(-v オプションと同じ)
debug        デバッグモードで実行(-d オプションと同じ)
concurrency  | 並列実行数(default: 1)
             | マルチパートアップロードの際のパートのアップロードの並列実行数となります。
tempDir      | 一時ファイルの保存先
             | 標準入力を使用したアップロードの場合は一時的にこのディレクトリの保存されます。
===========  ================================================================================

**[logging] セクション**

====  ===================================================
type  ログ出力の種類(none, file, stdout, stderr)
file  *file* タイプ時の出力先のファイルパス
====  ===================================================

**[storage] セクション**

==================  =============================================================================================
endpoint            IIJGIO ストレージ＆アナリシスサービスのStorage APIのエンドポイント
accessKeyId         APIのアクセスキーID
secretAccessKey     APIのシークレットアクセスキー
secure              SSL/TLSプロトコルを用いた通信の暗号化(HTTPS)を使用するかどうか(true,false)
multipartChunkSize  | マルチパートアップロードのチャンクサイズ(Bytes)。
                    | アップロードするファイルが指定のサイズより大きい場合にはマルチパートアップロードとなり、
                      このサイズで分割してアップロードします。このサイズを下回る場合にはPUT Objectとなります。
retry               | HTTP/HTTPS リクエスト失敗時のリトライ回数 (デフォルト: 2)
                    | リトライしない場合は 0 を指定してください。
retryInterval       | リトライを実施する間隔（ミリ秒単位, デフォルト: 3000）
                    | 1秒 = 1000 となります。
abortOnFailure      | マルチパートアップロードを使用したアップロードに失敗した場合に、該当のマルチパートアップロ
                      ードを削除するかどうか(true,false)
                    | マルチパートアップロードの残留を防ぐ場合に有効にしてください。マルチパートアップロードを再
                      開する場合は false を指定してください。
==================  =============================================================================================


設定例
======

::

   [dagtools]
   debug = false
   verbose = true
   proxy =
   concurrency = 2
   tempDir = /var/tmp

   [logging]
   type = file
   file = dagtools.log

   [storage]
   endpoint = storage-dag.iijgio.com
   accessKeyId = <Access Key Id>
   secretAccessKey = <Secret Access Key>
   secure = true
   multipartChunkSize = 1073741824 # 1GB
   retry = 2 # number of retries
   retryInterval = 3000 # 3.0 seconds
   abortOnFailure = true

| <Access Key Id> および <Secret Access Key> はサービスオンラインより払い出されたアクセスキーIDとシークレットアクセスキーとなります。
| 認証情報が含まれるファイルですので、ファイルの権限設定など適切に行ってください。


使い方
======
::

   Usage:
     dagtools [-h] [-d] [-v] [-f <config file>] <command> [<args>]
   
   Options:
     -d    debug mode
     -f string
           specify an alternate configuration file (default: ./dagtools.ini or /etc/dagtools.ini)
     -h    print a help message and exit
     -v    verbose mode
     -version
           show version
   
   Commands:
             ls: list buckets or objects
            cat: get an object and print to standard output
            get: get an object and write to a file
          exist: check to exist buckets/objects
             rm: delete a bucket or object[s]
            put: put a bucket or object[s]
           help: print a command usage
           sync: synchronize with objects on DAG storage and local files
         policy: manage a bucket policy (put, cat, rm)
          space: display used storage space
        traffic: display network traffics
        uploads: manage multipart-upload[s]

実行例
======

バケット新規作成(PUT Bucket)
----------------------------
::

   $ dagtools put mybucket


ファイルのアップロード(PUT Object)
----------------------------------
単一のファイルをアップロード::

  $ dagtools put path/to/file mybucket:foo/bar/my-object
  or
  $ dagtools put mybucket:foo/bar/my-object < path/to/file

複数のファイルをアップロード::

  $ dagtools put path/to/file1 path/to/file2 mybucket:foo/bar/
  $ dagtools put path/to/file* mybucket:foo/bar/

ディレクトリを指定してアップロード::

  # 以下の場合はオブジェクトキーにディレクトリ名を含みます (foo/bar/dir/...)
  $ dagtools put -r path/to/dir/ mybucket:foo/bar/

  # 以下の場合はオブジェクトキーにディレクトリ名は含みません (foo/bar/...)
  $ dagtools put -r path/to/dir/ mybucket:foo/bar

マルチパートアップロードの再開::

  $ dagtools put -upload-id=E-Ckgc1u-fAEIhDcPYcx430ygDjDq1IO7zILJF9W1HpUrbjq3UVlbV23UA45UFNS9nocgth7vsOh.zWaqGm.Jg-UGRiX6WCBPvNM_teEwa4- path/to/file mybucket:foo/bar/my-object


オブジェクトの取得(GET Object)
------------------------------
カレントディレクトリに書き出す::

  $ dagtools get mybucket:foo/bar/my-object

.. note::

   上記の例では *my-object* というファイル名でカレントディレクトリに保存されます。

書き込み先を指定して書き出す::

  $ dagtools get mybucket:foo/bar/my-object path/to/file

ディレクトリを指定して一括で取得する::

  $ dagtools get -r mybucket:foo/bar/dir/ path/to/directory/

.. note::

   - 末尾にスラッシュを付けた場合には、そのディレクトリにサブディレクトリを作成します。(上記の例では :code:`path/to/directory/dar/` が作られます)
   - 逆に付けなかった場合には、そのディレクトリ名に置き換えられます。(上記の例では :code:`foo/bar/dir/` は :code:`path/to/directory/` として格納します)

オブジェクトの内容を表示(標準出力)::

  $ dagtools cat mybucket:foo/bar/my-object


バケットの削除(DELETE Bucket)
-----------------------------
空のバケットを削除::

   $ dagtools rm mybucket

.. note::

   オブジェクトが1つ以上存在する場合は削除されません。

バケット内の全てオブジェクトも一緒に削除::

  $ dagtools rm -r mybucket


オブジェクトの削除(DELETE Objects)
----------------------------------
単一のオブジェクトを削除::

  $ dagtools rm mybucket:foo/bar/my-object

ディレクトリを指定して削除::

  $ dagtools rm -r mybucket:foo/bar/

ファイルのプレフィックスを指定して削除::

  $ dagtools rm "mybucket:foo/bar/my-*"

.. note::

   | Linux環境の場合はアスタリスク( `*` )はシェルのワイルドカードとしてファイルパスに展開されてコマンドを実行される場合があります。
   | 従って、ダブルクオート( `"` )もしくはシングルクォート( `'` )で囲んで指定してください。

削除したオブジェクトを表示::

  $ dagtools rm -v -r mybucket:foo/


バケットの一覧表示(List Buckets)
--------------------------------
::

   $ dagtools ls


オブジェクトの一覧表示(List Objects)
------------------------------------
ルートディレクトリのオブジェクト一覧::

  $ dagtools ls mybucket

指定したディレクトリのオブジェクト一覧::

  $ dagtools ls mybucket:foo
  or
  $ dagtools ls mybucket:foo/

オブジェクトのサイズを読みやすい形式(Human-Readable)で表示::

  $ dagtools ls -h mybucket:foo

サブディレクトリを再帰的に表示::

  $ dagtools ls -r mybucket:foo

指定したファイルのプレフィックスに一致するオブジェクトのみ表示::

  $ dagtools ls "mybucket:foo/bar/my*"

.. note::

   | 末尾にアスタリスク( `*` ) を付けることでオブジェクトキーに前方一致するオブジェクトの一覧を表示します。
   | Linux環境の場合、アスタリスク( `*` )はシェルのワイルドカードとしてファイルパスに展開されてコマンドを実行される場合がありますので、従って、ダブルクオート( `"` )もしくはシングルクォート( `'` )で囲んで指定してください。

TSV形式で表示::

  $ dagtools ls -tsv mybucket:foo

JSON形式で表示::

  $ dagtools ls -json mybucket:foo

ETagも含めてリストを表示する::

  $ dagtools ls -etag mybucket:foo


マルチパートアップロードの一覧表示(List uploads)
------------------------------------------------
バケット名とオブジェクトキーに一致するマルチパートアップロード一覧::

  $ dagtools uploads ls mybucket:foo

バケットの全てのマルチパートアップロードを表示::

  $ dagtools uploads ls -r mybucket


進行中のマルチパートアップロードの削除(DELETE uploads)
------------------------------------------------------
バケット名とオブジェクトキーに一致する進行中のマルチパートアップロードの削除::

  $ dagtools uploads rm mybucket:foo

アップロードIDを指定して削除::

  $ dagtools uploads rm mybucket:foo:E-Ckgc1u-fAEIhDcPYcx430ygDjDq1IO7zILJF9W1HpUrbjq3UVlbV23UA45UFNS9nocgth7vsOh.zWaqGm.Jg-UGRiX6WCBPvNM_teEwa4-

バケット内の全ての進行中のマルチパートアップロードを削除::

  $ dagtools uploads rm -r mybucket


ディレクトリを同期
------------------

- ローカルのディレクトリとDAGストレージのバケット/ディレクトリ間で片方向の同期を行う機能です。
- ファイルの更新日時とサイズを元に変更を検出し、2回目以降の同期は変更されているファイルのみアップロード/ダウンロードします。

.. note::

   注意: ファイルの削除は同期されません

DAGストレージからローカルのディレクトリに同期::

  $ dagtools sync mybucket:foo/bar/ /path/to/local-dir/

ローカルのディレクトリからDAGストレージに同期::

  $ dagtools sync /path/to/local-dir/ mybucket:foo/bar/

同期の状況を表示::

  $ dagtools -v sync /path/to/local-dir/ mybucket:foo/bar/

確認(dry-run)::

  $ dagtools -v sync -n /path/to/local-dir/ mybucket:foo/bar/


バケットポリシーの登録(PUT Bucket policy)
-----------------------------------------
::

   $ dagtools policy put mybucket policy.json
   or
   $ dagtools policy put mybucket < policy.json


バケットポシリーの取得(GET Bucket policy)
-----------------------------------------
標準出力に表示::

   $ dagtools policy cat mybucket


バケットポリシーの削除(DELETE Bucket policy)
--------------------------------------------
::

   $ dagtools policy rm mybucket


ストレージ使用量の取得(GET Service space)
-----------------------------------------
::

   $ dagtools space

ストレージに対するネットワーク通信量の取得(GET Service traffic)
---------------------------------------------------------------

日付を指定して取得::

  $ dagtools traffic 20150401

先月１日から今日までの一覧を取得する::

  $ dagtools traffic -b 1


バケットまたはオブジェクトの存在確認(HEAD Bucket, HEAD Object)
--------------------------------------------------------------

バケットの存在確認::

  $ dagtools exist mybucket

オブジェクトの存在確認::

  $ dagtools exist mybucket:foo

.. note::

   - 終了ステータスで結果を確認することができます。
       - 存在する場合: ``0``
       - 存在しない場合(エラー): ``1``
   - 表示オプション(``-v``)が有効の場合は標準出力に結果を表示します。
   - コマンド引数に複数のバケットまたはオブジェクトを指定することもできます。


その他
======

終了ステータスについて
----------------------

終了ステータスはdagtools返されたサービスからのレスポンス内容を反映します。

.. note::

   - 終了ステータス
      - レスポンスが2xxだった場合: ``0``
      - レスポンスが4xx又は5xxだった場合: ``1``

終了ステータスは、一般的な方法でスクリプトから参照する事が可能です。
たとえばWindowsであれば環境変数 %ERRORLEVEL% を参照する事で、またLinuxであれば $? を参照する事で値を確認できます。

エラー時のメッセージについて
----------------------------

dagtoolsが何らかのエラーを受け取った場合、以下のフォーマットで標準エラー出力にメッセージが出力されます。

.. note::

   [Error] <レスポンスコード> <メッセージ> (<リクエストID>)

- レスポンスコードは、受信したHTTPレスポンスコードです。
- メッセージは、発生した問題を記述したメッセージです。
- リクエストIDは個々のリクエストに付与される識別子で、サポートへのお問い合わせの際にお知らせ頂くものです。

